# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for the go-trust project.

## What is an ADR?

An Architecture Decision Record (ADR) is a document that captures an important architectural decision made along with its context and consequences.

## ADR Format

We use the [MADR (Markdown Any Decision Records)](https://adr.github.io/madr/) format for our ADRs.

## Index of ADRs

- [ADR-0000](0000-use-markdown-architectural-decision-records.md) - Use Markdown Architectural Decision Records
- [ADR-0001](0001-pipeline-architecture.md) - Pipeline Architecture with YAML Configuration
- [ADR-0002](0002-configuration-system.md) - Hierarchical Configuration System
- [ADR-0003](0003-concurrent-processing.md) - Concurrent TSL Processing with Worker Pools
- [ADR-0004](0004-xslt-transformation.md) - XSLT Transformation with libxslt via CGO
- [ADR-0005](0005-api-design.md) - API Design with AuthZEN and Gin Framework
- [ADR-0006](0006-error-handling.md) - Error Handling Strategy
- [ADR-0007](0007-observability.md) - Observability with Prometheus and Health Checks

## Status Definitions

- **Accepted**: The decision has been made and is currently in effect
- **Superseded**: The decision has been replaced by a newer decision
- **Deprecated**: The decision is no longer recommended but still in use
- **Proposed**: The decision is under consideration

## Creating a New ADR

1. Copy the template from `template.md`
2. Name it with the next sequential number: `NNNN-title-with-dashes.md`
3. Fill in all sections
4. Update this README with a link to the new ADR
5. Submit a pull request

## References

- [ADR GitHub Organization](https://adr.github.io/)
- [MADR Format](https://adr.github.io/madr/)
- [When to Write an ADR](https://github.com/joelparkerhenderson/architecture-decision-record#when-to-write-an-adr)
