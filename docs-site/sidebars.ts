import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  tutorialSidebar: [
    'intro',
    {
      type: 'category',
      label: 'Getting Started',
      items: [
        'getting-started/installation',
        'getting-started/quick-start',
      ],
    },
    {
      type: 'category',
      label: 'Guides',
      items: [
        'guides/workspaces',
        'guides/session-management',
        'guides/ai-workflows',
        'guides/hooks',
      ],
    },
    {
      type: 'category',
      label: 'Reference',
      items: [
        'reference/commands',
        'reference/configuration',
        'reference/mcp-api',
      ],
    },
  ],
};

export default sidebars;