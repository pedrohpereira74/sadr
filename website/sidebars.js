// @ts-check

/** @type {import('@docusaurus/plugin-content-docs').SidebarsConfig} */
const sidebars = {
  docs: [
    'intro',
    'getting-started',
    {
      type: 'category',
      label: 'Guide',
      collapsed: false,
      items: [
        'guide/configuration',
        'guide/usage',
        'guide/ai-features',
        'guide/team-workflows',
      ],
    },
    {
      type: 'category',
      label: 'Reference',
      collapsed: false,
      items: [
        'reference/cli',
        'reference/config-schema',
        'reference/record-format',
      ],
    },
    {
      type: 'category',
      label: 'Architecture',
      collapsed: true,
      items: [
        'architecture/overview',
      ],
    },
  ],
};

module.exports = sidebars;
