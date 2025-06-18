# ADR-028: Documentation Site and Hosting

**Status**: Accepted
**Date**: 2025-06-19

## Context

As Amux approaches its v0.1.0 release, we need proper documentation to help users understand and adopt the tool. The existing README.md has grown to nearly 400 lines, making it overwhelming for new users and difficult to navigate for specific information.

### Current Documentation Issues

1. **README Overload**: The README contains everything from installation to advanced configuration
2. **Poor Discoverability**: Users can't easily find specific topics
3. **No Version Management**: Can't maintain docs for different versions
4. **Limited Formatting**: Markdown files in the repo lack navigation and search
5. **No Interactive Elements**: Can't provide live examples or interactive demos

### Documentation Requirements

1. **Quick Onboarding**: New users should understand Amux in under 5 minutes
2. **Comprehensive Guides**: Detailed documentation for all features
3. **API Reference**: Complete MCP API documentation for AI agent developers
4. **Searchable**: Users should find information quickly
5. **Maintainable**: Easy to update as the project evolves
6. **Free Hosting**: No ongoing costs for an open-source project

## Decision

We will use **Docusaurus** for documentation and host it on **GitHub Pages**.

### Documentation Framework: Docusaurus

Docusaurus provides:

- Modern React-based static site generator
- Built-in search functionality
- Versioning support for future releases
- Dark mode (important for developer tools)
- MDX support for interactive components
- Excellent performance and SEO

### Hosting: GitHub Pages

GitHub Pages offers:

- Free hosting for open-source projects
- Automatic deployment via GitHub Actions
- No maintenance overhead
- Direct integration with the repository
- URL: `https://choplin.github.io/amux/`

### Documentation Structure

```text
docs-site/
├── docs/
│   ├── intro.md                    # Welcome page
│   ├── getting-started/
│   │   ├── installation.md
│   │   └── quick-start.md
│   ├── guides/
│   │   ├── workspaces.md
│   │   ├── session-management.md
│   │   ├── ai-workflows.md
│   │   └── hooks.md
│   └── reference/
│       ├── commands.md
│       ├── configuration.md
│       └── mcp-api.md
├── src/
│   ├── components/              # React components
│   ├── css/                     # Custom styles
│   └── pages/                   # Additional pages
└── static/
    └── img/                     # Images and assets
```

### README Simplification

The main README.md will be reduced to ~120 lines containing only:

- Project tagline and value proposition
- Installation instructions
- 5-step quick start
- Links to full documentation

## Rationale

### Why Docusaurus?

1. **Developer-Friendly**: Built for technical documentation
2. **Feature-Rich**: Search, versioning, i18n ready
3. **Low Barrier**: Markdown-based with optional React enhancements
4. **Active Community**: Well-maintained by Meta
5. **Static Output**: Can be hosted anywhere

### Why GitHub Pages?

1. **Zero Cost**: Free for public repositories
2. **Zero Configuration**: GitHub Actions handle deployment
3. **High Reliability**: Backed by GitHub's infrastructure
4. **Version Control**: Documentation versions tied to git tags
5. **Single Platform**: Everything stays within GitHub

### Alternative Hosting Considered

- **Vercel/Netlify**: Better features but requires separate account
- **Custom Domain**: Would require purchasing and maintaining a domain
- **Wiki**: Too limited for comprehensive documentation
- **GitBook**: Proprietary and has limitations in free tier

## Consequences

### Positive

- **Improved User Experience**: Clean, navigable documentation
- **Better Onboarding**: New users can start quickly
- **Maintainability**: Easier to update and version documentation
- **Professional Appearance**: Proper docs site increases project credibility
- **SEO Benefits**: Better discoverability for the project
- **Community Contributions**: Easier for others to contribute docs

### Negative

- **Build Complexity**: Adds Node.js build step to the project
- **GitHub Pages URL**: Subpath URL (`/amux/`) requires baseUrl configuration
- **Limited Customization**: Bound by GitHub Pages limitations
- **Separate Codebase**: Documentation lives in a subdirectory

### Migration Path

1. Documentation is already written and organized in `docs-site/`
2. GitHub Actions workflow exists in `.github/workflows/deploy-docs.yml`
3. On merge to main, documentation will auto-deploy
4. No manual intervention required

## Implementation Notes

The implementation includes:

1. **Docusaurus Configuration**: TypeScript-based config with dark mode default
2. **GitHub Actions**: Automated build and deploy on push to main
3. **Visual Assets**: Custom logo, hero image, and social cards
4. **Content Migration**: All documentation moved from README to appropriate guides
5. **URL Configuration**: Set up for `https://choplin.github.io/amux/`

## References

- Docusaurus Documentation: <https://docusaurus.io/>
- GitHub Pages Documentation: <https://docs.github.com/pages>
- Issue #6: Documentation improvements
