import type {ReactNode} from 'react';
import clsx from 'clsx';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

type FeatureItem = {
  title: string;
  Svg: React.ComponentType<React.ComponentProps<'svg'>>;
  description: ReactNode;
};

const FeatureList: FeatureItem[] = [
  {
    title: 'Instant Isolation',
    Svg: require('@site/static/img/features/instant-isolation.svg').default,
    description: (
      <>
        Create isolated Git worktree environments in seconds, not minutes.
        No Docker, no VMs, just pure Git efficiency.
      </>
    ),
  },
  {
    title: 'True Parallel Development',
    Svg: require('@site/static/img/features/parallel-development.svg').default,
    description: (
      <>
        Run multiple AI agents simultaneously without conflicts.
        Each agent works in its own workspace with zero interference.
      </>
    ),
  },
  {
    title: 'CLI and MCP Integration',
    Svg: require('@site/static/img/features/seamless-integration.svg').default,
    description: (
      <>
        Works with CLI tools like fzf and ripgrep.
        Full MCP support for AI assistants like Claude Code.
      </>
    ),
  },
];

function Feature({title, Svg, description}: FeatureItem) {
  return (
    <div className={clsx('col col--4')}>
      <div className="text--center">
        <Svg className={styles.featureSvg} role="img" />
      </div>
      <div className="text--center padding-horiz--md">
        <Heading as="h3">{title}</Heading>
        <p>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): ReactNode {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
        

        <div className="row margin-top--lg">
          <div className="col col--12">
            <Heading as="h2" className="text--center">Quick Example</Heading>
            <div className={styles.codeExample}>
              <pre>
                <code>{`# Create isolated workspaces for parallel development
amux ws create feat-auth
amux ws create fix-security
amux ws create docs-update

# Run different AI agents in each workspace
amux run claude --workspace feat-auth
amux run gpt --workspace fix-security
amux run gemini --workspace docs-update

# Monitor all agents
amux ps
┌─────────┬────────┬───────────────┬────────┐
│ SESSION │ AGENT  │ WORKSPACE     │ STATUS │
├─────────┼────────┼───────────────┼────────┤
│ sess-1a │ claude │ feat-auth     │ 🟢 busy │
│ sess-2b │ gpt    │ fix-security  │ 🟢 busy │
│ sess-3c │ gemini │ docs-update   │ 🟢 busy │
└─────────┴────────┴───────────────┴────────┘`}</code>
              </pre>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}