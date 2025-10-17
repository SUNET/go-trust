# Use Markdown Architectural Decision Records

- Status: Accepted
- Deciders: Development Team
- Date: 2025-10-17

## Context and Problem Statement

We need to document architectural decisions made in the go-trust project to help current and future developers understand why certain choices were made. How should we capture and maintain this architectural knowledge?

## Decision Drivers

- Need to preserve architectural knowledge and reasoning
- Documentation should be version-controlled alongside code
- Format should be readable and writable by all team members
- Should integrate well with existing development workflow
- Need to track decisions over time as they evolve

## Considered Options

- Wiki pages (GitHub Wiki, Confluence)
- Markdown files in the repository
- Google Docs or other external documentation
- Code comments only
- No formal documentation

## Decision Outcome

Chosen option: "Markdown files in the repository" using the MADR (Markdown Any Decision Records) format, because it keeps documentation close to the code, is version-controlled, and is easily accessible to all developers.

### Positive Consequences

- ADRs are version-controlled with the code
- Easy to review in pull requests
- Searchable and browsable
- Markdown is familiar to all developers
- Can be referenced in code comments and issues
- Supports incremental documentation

### Negative Consequences

- Requires discipline to keep updated
- May accumulate outdated decisions if not maintained
- No fancy formatting compared to wiki tools

## Pros and Cons of the Options

### Wiki pages

- Good, because wiki tools offer rich formatting
- Good, because easy to organize and cross-link
- Bad, because separate from code repository
- Bad, because not reviewed in pull requests
- Bad, because can become stale

### Markdown files in the repository

- Good, because version-controlled with code
- Good, because reviewed in pull requests
- Good, because always in sync with codebase
- Good, because simple text format
- Good, because easy to search
- Bad, because requires manual indexing

### Google Docs or external documentation

- Good, because rich formatting options
- Good, because collaborative editing
- Bad, because separate from code
- Bad, because access control complexity
- Bad, because not version-controlled

### Code comments only

- Good, because right next to implementation
- Bad, because scattered across codebase
- Bad, because no high-level overview
- Bad, because difficult to find historical decisions

### No formal documentation

- Good, because no overhead
- Bad, because knowledge is lost
- Bad, because new developers lack context
- Bad, because decisions are forgotten

## Links

- [MADR Format](https://adr.github.io/madr/)
- [ADR GitHub Organization](https://adr.github.io/)
