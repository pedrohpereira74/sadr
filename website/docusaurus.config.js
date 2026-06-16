// @ts-check
const { themes } = require('prism-react-renderer');

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'sadr',
  tagline: 'Capture code with context. Because snippets without a "why" are sadr.',
  // Update these when deploying to GitHub Pages
  url: 'https://pedrohpereira74.github.io',
  baseUrl: '/sadr/',

  organizationName: 'pedrohpereira74',
  projectName: 'sadr',

  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          sidebarPath: require.resolve('./sidebars.js'),
          routeBasePath: '/',
          editUrl: 'https://github.com/pedrohpereira74/sadr/tree/main/website/',
        },
        blog: false,
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      navbar: {
        title: 'sadr',
        items: [
          {
            type: 'docSidebar',
            sidebarId: 'docs',
            position: 'left',
            label: 'Docs',
          },
          {
            href: 'https://github.com/pedrohpereira74/sadr/releases',
            label: 'Releases',
            position: 'right',
          },
          {
            href: 'https://github.com/pedrohpereira74/sadr',
            label: 'GitHub',
            position: 'right',
          },
        ],
      },
      footer: {
        style: 'dark',
        links: [
          {
            title: 'Docs',
            items: [
              { label: 'Getting Started', to: '/getting-started' },
              { label: 'CLI Reference', to: '/reference/cli' },
              { label: 'Config Schema', to: '/reference/config-schema' },
            ],
          },
          {
            title: 'Project',
            items: [
              { label: 'GitHub', href: 'https://github.com/pedrohpereira74/sadr' },
              { label: 'Releases', href: 'https://github.com/pedrohpereira74/sadr/releases' },
              { label: 'Issues', href: 'https://github.com/pedrohpereira74/sadr/issues' },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} sadr. Built with Docusaurus.`,
      },
      prism: {
        theme: themes.github,
        darkTheme: themes.dracula,
        additionalLanguages: ['bash', 'yaml', 'go'],
      },
      colorMode: {
        defaultMode: 'dark',
        disableSwitch: false,
      },
    }),
};

module.exports = config;
