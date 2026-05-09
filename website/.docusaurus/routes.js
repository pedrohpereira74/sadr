import React from 'react';
import ComponentCreator from '@docusaurus/ComponentCreator';

export default [
  {
    path: '/sadr/__docusaurus/debug',
    component: ComponentCreator('/sadr/__docusaurus/debug', '4ed'),
    exact: true
  },
  {
    path: '/sadr/__docusaurus/debug/config',
    component: ComponentCreator('/sadr/__docusaurus/debug/config', '23d'),
    exact: true
  },
  {
    path: '/sadr/__docusaurus/debug/content',
    component: ComponentCreator('/sadr/__docusaurus/debug/content', '890'),
    exact: true
  },
  {
    path: '/sadr/__docusaurus/debug/globalData',
    component: ComponentCreator('/sadr/__docusaurus/debug/globalData', '099'),
    exact: true
  },
  {
    path: '/sadr/__docusaurus/debug/metadata',
    component: ComponentCreator('/sadr/__docusaurus/debug/metadata', '4cd'),
    exact: true
  },
  {
    path: '/sadr/__docusaurus/debug/registry',
    component: ComponentCreator('/sadr/__docusaurus/debug/registry', '75b'),
    exact: true
  },
  {
    path: '/sadr/__docusaurus/debug/routes',
    component: ComponentCreator('/sadr/__docusaurus/debug/routes', '65a'),
    exact: true
  },
  {
    path: '/sadr/',
    component: ComponentCreator('/sadr/', '725'),
    routes: [
      {
        path: '/sadr/',
        component: ComponentCreator('/sadr/', '062'),
        routes: [
          {
            path: '/sadr/',
            component: ComponentCreator('/sadr/', '710'),
            routes: [
              {
                path: '/sadr/architecture/overview',
                component: ComponentCreator('/sadr/architecture/overview', 'd48'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/sadr/getting-started',
                component: ComponentCreator('/sadr/getting-started', '1f3'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/sadr/guide/ai-features',
                component: ComponentCreator('/sadr/guide/ai-features', '4f6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/sadr/guide/configuration',
                component: ComponentCreator('/sadr/guide/configuration', '974'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/sadr/guide/team-workflows',
                component: ComponentCreator('/sadr/guide/team-workflows', 'c08'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/sadr/guide/usage',
                component: ComponentCreator('/sadr/guide/usage', 'cd0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/sadr/reference/cli',
                component: ComponentCreator('/sadr/reference/cli', '23c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/sadr/reference/config-schema',
                component: ComponentCreator('/sadr/reference/config-schema', '620'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/sadr/reference/record-format',
                component: ComponentCreator('/sadr/reference/record-format', '2f4'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/sadr/',
                component: ComponentCreator('/sadr/', 'c0d'),
                exact: true,
                sidebar: "docs"
              }
            ]
          }
        ]
      }
    ]
  },
  {
    path: '*',
    component: ComponentCreator('*'),
  },
];
