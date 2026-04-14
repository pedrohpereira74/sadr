package cmd

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pedrohpereira74/sadr/internal/config"
	jiraclient "github.com/pedrohpereira74/sadr/internal/jira"
	"github.com/pedrohpereira74/sadr/internal/ui"
)

func runDisableJiraWarning() {
	if err := saveJiraGlobalConfig(func(cfg *config.JiraConfig) {
		cfg.DisableProjectWarning = true
	}); err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to save config: %v", err))
		return
	}
	ui.Success(os.Stderr, "jira project warning disabled.")
}

func runSetupJira() {
	method := runSelect("authentication method:", []selectOption{
		{Label: "basic auth (username + password)", Value: "basic"},
		{Label: "bearer / pat (personal access token)", Value: "bearer"},
		{Label: "oauth 1.0a (rsa-sha1, requires admin setup)", Value: "oauth"},
	})
	if method == "" {
		return
	}

	switch method {
	case "basic":
		runSetupJiraBasic()
	case "bearer":
		runSetupJiraBearer()
	case "oauth":
		runSetupJiraOAuth()
	}
}

func runSetupJiraBasic() {
	ui.Info(os.Stderr, "basic auth uses your ldap/ad username and password.")
	ui.Info(os.Stderr, "check with your jira admin whether basic auth is enabled for api access.")
	if !confirmPromptFn("continue with basic auth?") {
		return
	}

	username := strings.TrimSpace(runTextarea("jira username (email or ldap username):", "you@company.com"))
	if username == "" {
		return
	}
	password := strings.TrimSpace(runTextarea("jira password:", ""))
	if password == "" {
		return
	}

	if err := saveJiraGlobalConfig(func(cfg *config.JiraConfig) {
		cfg.Username = username
		cfg.Password = password
	}); err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to save config: %v", err))
		return
	}
	ui.Success(os.Stderr, "saved.")
}

func runSetupJiraBearer() {
	ui.Info(os.Stderr, "bearer tokens (personal access tokens) require jira server 8.14+ or data center.")
	ui.Info(os.Stderr, "check with your jira admin whether your version supports pats and how to generate one.")
	if !confirmPromptFn("continue with bearer token?") {
		return
	}

	token := strings.TrimSpace(runTextarea("personal access token:", ""))
	if token == "" {
		return
	}
	if len(token) < 20 {
		ui.Warning(os.Stderr, fmt.Sprintf("this token is only %d characters — jira pats are usually 24 or more. double-check you copied the full token.", len(token)))
		if !confirmPromptFn("continue anyway?") {
			return
		}
	}

	if err := saveJiraGlobalConfig(func(cfg *config.JiraConfig) {
		cfg.Token = token
	}); err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to save config: %v", err))
		return
	}
	ui.Success(os.Stderr, "saved.")
}

func runSetupJiraOAuth() {
	ui.Info(os.Stderr, "oauth 1.0a requires your admin to register sadr as an application link in jira.")
	ui.Info(os.Stderr, "ask your admin for the private key file (.pem) and the consumer key.")
	if !confirmPromptFn("continue with oauth 1.0a?") {
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to get home directory: %v", err))
		return
	}
	jiraDir := filepath.Join(home, ".sadr", "jira")
	_ = os.MkdirAll(jiraDir, 0700)

	pemPath, err := findSinglePEM(jiraDir)
	if err != nil {
		ui.Error(os.Stderr, err.Error())
		return
	}
	if pemPath == "" {
		ui.Info(os.Stderr, fmt.Sprintf("no private key found in %s/", jiraDir))
		ui.Info(os.Stderr, "ask your admin for the team's private key file (.pem) and place it in that directory, then run this command again.")
		return
	}

	jiraURL := strings.TrimRight(strings.TrimSpace(runTextarea("jira url:", "https://jira.yourcompany.com")), "/")
	if jiraURL == "" {
		return
	}
	consumerKey := strings.TrimSpace(runTextarea("consumer key (provided by your admin):", "sadr-cli"))
	if consumerKey == "" {
		return
	}

	ui.Info(os.Stderr, "requesting oauth token from jira...")

	callbackPort, err := freePort()
	if err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to find free port: %v", err))
		return
	}
	callbackURL := fmt.Sprintf("http://localhost:%d/callback", callbackPort)

	privKey, err := loadRSAKey(pemPath)
	if err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to load private key: %v", err))
		return
	}

	requestToken, err := getRequestToken(jiraURL, consumerKey, callbackURL, privKey)
	if err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to get request token: %v", err))
		ui.Info(os.Stderr, "make sure the admin has registered sadr and the consumer key matches.")
		return
	}

	authURL := fmt.Sprintf("%s/plugins/servlet/oauth/authorize?oauth_token=%s", jiraURL, url.QueryEscape(requestToken))
	fmt.Fprintf(os.Stderr, "\nopening browser to authorize sadr:\n%s\n\n", authURL)
	openBrowser(authURL)

	ui.Info(os.Stderr, "waiting for authorization in the browser...")
	verifier, err := waitForCallback(callbackPort)
	if err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("authorization failed: %v", err))
		return
	}

	ui.Info(os.Stderr, "exchanging token...")
	accessToken, accessTokenSecret, err := getAccessToken(jiraURL, consumerKey, requestToken, verifier, privKey)
	if err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to get access token: %v", err))
		return
	}

	if err := saveJiraGlobalConfig(func(cfg *config.JiraConfig) {
		cfg.ConsumerKey = consumerKey
		cfg.PrivateKeyPath = pemPath
		cfg.AccessToken = accessToken
		cfg.AccessTokenSecret = accessTokenSecret
	}); err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to save config: %v", err))
		return
	}

	ui.Success(os.Stderr, fmt.Sprintf("jira connected. testing connection to %s...", jiraURL))

	c := jiraclient.NewClientFromConfig(jiraclient.ClientConfig{
		BaseURL:           jiraURL,
		ConsumerKey:       consumerKey,
		PrivateKeyPath:    pemPath,
		AccessToken:       accessToken,
		AccessTokenSecret: accessTokenSecret,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = c.FetchAll(ctx, []string{"SADR-TEST-NONEXISTENT"})
	if err != nil && !strings.Contains(err.Error(), "400") && !strings.Contains(err.Error(), "404") {
		ui.Info(os.Stderr, fmt.Sprintf("connection test inconclusive: %v", err))
	} else {
		ui.Success(os.Stderr, "connection ok. jira is ready to use.")
	}
}

func runSetupJiraAdmin() {
	home, err := os.UserHomeDir()
	if err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to get home directory: %v", err))
		return
	}

	adminDir := filepath.Join(home, ".sadr", "jira-admin")
	_ = os.MkdirAll(adminDir, 0700)
	privKeyPath := filepath.Join(adminDir, "jira_rsa.pem")
	pubKeyPath := filepath.Join(adminDir, "jira_rsa_pub.pem")

	if _, err := os.Stat(privKeyPath); err == nil {
		ui.Info(os.Stderr, "a private key already exists in ~/.sadr/jira-admin/.")
		ui.Info(os.Stderr, "regenerating requires updating the application link in jira and redistributing the new key to the entire team.")
		if !confirmPromptFn("regenerate the key pair?") {
			return
		}
	}

	if err := generateRSAKeyPair(privKeyPath, pubKeyPath); err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to generate rsa key pair: %v", err))
		return
	}

	pubPEM, _ := os.ReadFile(pubKeyPath)
	fmt.Fprintf(os.Stderr, "\npublic key (give this to your jira admin):\n\n%s\n", string(pubPEM))
	ui.Info(os.Stderr, "the admin should register sadr as an application link in:")
	ui.Info(os.Stderr, "  jira admin → applications → application links → create link")
	ui.Info(os.Stderr, "  type: generic application | incoming auth: oauth | consumer key: (choose a name, e.g. sadr-cli)")
	ui.Info(os.Stderr, "  public key: (paste the key above)")
	fmt.Fprintln(os.Stderr)
	ui.Warning(os.Stderr, "the private key at ~/.sadr/jira-admin/jira_rsa.pem is a team secret.")
	ui.Warning(os.Stderr, "distribute it only through secure internal channels (e.g. 1password, vault).")
	ui.Warning(os.Stderr, "never commit it to git or share it over email or chat.")
	fmt.Fprintln(os.Stderr)

	jiraDir := filepath.Join(home, ".sadr", "jira")
	_ = os.MkdirAll(jiraDir, 0700)
	destPath := filepath.Join(jiraDir, "jira_rsa.pem")

	if err := copyFile(privKeyPath, destPath); err != nil {
		ui.Error(os.Stderr, fmt.Sprintf("failed to copy key to ~/.sadr/jira/: %v", err))
		return
	}

	ui.Success(os.Stderr, fmt.Sprintf("key pair generated and saved to %s", adminDir))
	ui.Info(os.Stderr, fmt.Sprintf("private key also copied to %s", destPath))
	ui.Info(os.Stderr, "once the admin registers the application link in jira, run 'sadr config --setup-jira' to authenticate.")
}

func findSinglePEM(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", dir, err)
	}
	var found []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".pem") {
			found = append(found, filepath.Join(dir, e.Name()))
		}
	}
	switch len(found) {
	case 0:
		return "", nil
	case 1:
		return found[0], nil
	default:
		return "", fmt.Errorf("more than one .pem file found in %s — keep only your organization's key and run again", dir)
	}
}

func loadRSAKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return jiraclient.ParseRSAPrivateKey(data)
}

func generateRSAKeyPair(privPath, pubPath string) error {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	privFile, err := os.OpenFile(privPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer privFile.Close()
	if err := pem.Encode(privFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}); err != nil {
		return err
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return err
	}
	pubFile, err := os.OpenFile(pubPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer pubFile.Close()
	return pem.Encode(pubFile, &pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func saveJiraGlobalConfig(apply func(*config.JiraConfig)) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	globalConfigPath := filepath.Join(home, ".sadr", "global-config.yaml")
	cfg, _ := config.LoadGlobalFromFile(globalConfigPath)
	cfg.Jira = config.JiraConfig{}
	apply(&cfg.Jira)
	return config.SaveGlobalConfig(globalConfigPath, cfg)
}

func getRequestToken(jiraURL, consumerKey, callbackURL string, privKey *rsa.PrivateKey) (string, error) {
	c := &jiraclient.Client{
		BaseURL:     jiraURL,
		HTTP:        &http.Client{Timeout: 10 * time.Second},
		ConsumerKey: consumerKey,
		PrivateKey:  privKey,
	}
	endpoint := strings.TrimRight(jiraURL, "/") + "/plugins/servlet/oauth/request-token"
	body, err := c.OAuthRequest(context.Background(), "POST", endpoint, url.Values{
		"oauth_callback": {callbackURL},
	})
	if err != nil {
		return "", err
	}
	vals, err := url.ParseQuery(body)
	if err != nil {
		return "", fmt.Errorf("failed to parse request token response: %w", err)
	}
	token := vals.Get("oauth_token")
	if token == "" {
		return "", fmt.Errorf("no oauth_token in response: %s", body)
	}
	return token, nil
}

func getAccessToken(jiraURL, consumerKey, requestToken, verifier string, privKey *rsa.PrivateKey) (string, string, error) {
	c := &jiraclient.Client{
		BaseURL:     jiraURL,
		HTTP:        &http.Client{Timeout: 10 * time.Second},
		ConsumerKey: consumerKey,
		PrivateKey:  privKey,
		AccessToken: requestToken,
	}
	endpoint := strings.TrimRight(jiraURL, "/") + "/plugins/servlet/oauth/access-token"
	body, err := c.OAuthRequest(context.Background(), "POST", endpoint, url.Values{
		"oauth_verifier": {verifier},
	})
	if err != nil {
		return "", "", err
	}
	vals, err := url.ParseQuery(body)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse access token response: %w", err)
	}
	token := vals.Get("oauth_token")
	secret := vals.Get("oauth_token_secret")
	if token == "" {
		return "", "", fmt.Errorf("no oauth_token in response: %s", body)
	}
	return token, secret, nil
}

func waitForCallback(port int) (string, error) {
	verifierCh := make(chan string, 1)
	errCh := make(chan error, 1)

	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			verifier := r.URL.Query().Get("oauth_verifier")
			if verifier == "" {
				errCh <- fmt.Errorf("no oauth_verifier in callback")
				http.Error(w, "missing oauth_verifier", http.StatusBadRequest)
				return
			}
			fmt.Fprintln(w, "authorization successful. you can close this tab and return to the terminal.")
			verifierCh <- verifier
		}),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	select {
	case v := <-verifierCh:
		_ = srv.Shutdown(context.Background())
		return v, nil
	case err := <-errCh:
		_ = srv.Shutdown(context.Background())
		return "", err
	case <-ctx.Done():
		_ = srv.Shutdown(context.Background())
		return "", fmt.Errorf("timed out waiting for browser authorization")
	}
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
