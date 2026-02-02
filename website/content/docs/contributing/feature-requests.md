---
title: Feature Requests
weight: 2
---

Have an idea for NetVantage? We'd love to hear it.

## Before Requesting

1. **Check the roadmap** -- your idea may already be planned. See the [Roadmap](/#roadmap) for upcoming phases.
2. **Search existing issues** -- check [open issues](https://github.com/HerbHall/netvantage/issues) for similar requests
3. **Review the requirements** -- detailed planning docs are in [docs/requirements/](https://github.com/HerbHall/netvantage/tree/main/docs/requirements)

## How to Request

Use the [Feature Request template](https://github.com/HerbHall/netvantage/issues/new?template=feature_request.md) on GitHub.

A good feature request includes:

- **Problem statement** -- what are you trying to accomplish?
- **Proposed solution** -- how do you envision this working?
- **Alternatives considered** -- have you tried other approaches?
- **Use case** -- who benefits and in what scenario?

## Plugin vs Core

NetVantage has a plugin architecture. Consider whether your request:

- **Fits as a plugin** -- self-contained functionality that extends the platform (e.g., a new notification channel, a custom discovery method, an integration with a third-party service)
- **Fits in core** -- fundamental platform capability that other features depend on (e.g., authentication, database changes, API framework modifications)

Plugin requests can often be implemented independently using the [Apache 2.0 licensed Plugin SDK](https://github.com/HerbHall/netvantage/tree/main/pkg/plugin).

## What Happens Next

1. A maintainer will review and label the request
2. Community discussion may follow to refine the idea
3. Accepted features are added to the appropriate milestone
4. You'll be credited in the issue when the feature ships
