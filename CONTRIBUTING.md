# Navidrome Contribution Guide

Navidrome has one maintainer and limited review capacity. We welcome contributions that fit the project and remain practical to maintain. A smaller number of focused, well-understood contributions is more useful than a large volume of code that needs extensive rework.

## Support and conduct

Use [Discussions → Q&A](https://github.com/navidrome/navidrome/discussions/categories/q-a) for support. Please follow the [Code of Conduct](CODE_OF_CONDUCT.md).

## Issues and ideas

Anyone may report a suspected bug. Search existing [open and closed issues](https://github.com/navidrome/navidrome/issues?q=is%3Aissue) and [discussions](https://github.com/navidrome/navidrome/discussions), and check the [documentation](https://www.navidrome.org/docs/) and [FAQ](https://www.navidrome.org/docs/faq/) first. Use the [bug report form](https://github.com/navidrome/navidrome/issues/new?template=bug_report.yml) for suspected defects and [Discussions → Ideas](https://github.com/navidrome/navidrome/discussions/categories/ideas) for new features or changes to intended behavior.

Describe what happened, what you expected, and why that expectation should hold. Include reproduction steps where possible, with a reference to documentation, an applicable API contract, or a previously working version. If the problem is intermittent, explain what you observed and tried. Remove credentials and private information from logs and configuration.

A reported problem may involve intended behavior, configuration, or a product decision. An open issue or a label does not establish that a change has been accepted. Report security vulnerabilities [privately through GitHub](https://github.com/navidrome/navidrome/security/advisories/new).

## Agree on the change before implementation

For code contributions beyond the small exceptions below, explain the problem, intended behavior, and smallest proposed scope in an existing issue or discussion. Wait for an explicit maintainer comment agreeing that the change should be pursued. Keep the discussion in one place. Silence, community support, and previous merged contributions do not replace that agreement.

Small documentation corrections and small, unambiguous fixes that restore established behavior may be submitted directly. Provide evidence of the defect. These exceptions do not cover disputed behavior, new product choices, dependencies, or broader refactoring. If the intended behavior is unclear, discuss it first.

For large or cross-cutting changes, agree on the design and a sequence of small, reviewable phases before implementation. Explain why the change belongs in the core, what new complexity or dependencies it introduces, and which simpler alternatives you considered. Consider existing configuration, APIs, plugins, or external tools where appropriate. Splitting a change into PRs does not remove the need to agree on the overall design and scope.

## Review your work before opening a PR

Before opening any pull request, including a draft, you must personally read and review the entire diff and the relevant surrounding code, understand the implementation, and verify the affected behavior. This applies when an AI agent writes the code or creates the PR for you. An agent's summary or review does not satisfy this requirement. Personally review each subsequent update and verify its effects before pushing it or asking an agent to push it.

Check the assumptions behind the change. Explain the implementation and important tradeoffs in your own words. For code changes, personally test the affected functionality, add regression tests where practical, and report the commands run and results actually observed. Identify failures, untested cases, and limitations. If an automated test is impractical, explain why and provide manual verification steps.

Stay involved during review: answer questions, investigate feedback, and revise the work. If you need more time or cannot continue, say so. The maintainer is not committing to finish an abandoned implementation.

## AI Disclosure

AI tools may assist professional and hobbyist contributors. All code contributors must be able to personally understand, review, test, and maintain their changes, including security and performance implications; no diploma, job title, or employment is required. If you cannot assess the code yourself, report the problem instead of submitting AI-generated code. Bug reports, documentation improvements, and translations remain welcome.

AI-assisted work must meet the personal full-diff review and functional testing requirements above before any PR is opened, including drafts or agent-created PRs. You remain responsible for explaining the implementation, addressing feedback, and correcting the code.

Declare whether AI was used to generate or modify submitted code or tests, draft substantive PR text, or create the PR. A simple Yes or No in the PR description is enough. Tool names, prompts, and transcripts are not required.

Write the final PR explanation and review replies in your own words. Do not send solely AI-generated responses to maintainers or other contributors.

## Keep the result maintainable

Prefer existing patterns and the smallest clear solution. Justify new abstractions, dependencies, configuration, and compatibility obligations. Include the documentation and tests needed to understand the behavior and change it safely later.

The maintainer must be able to understand and maintain the result without depending on your future availability. An offer to provide ongoing support does not replace this requirement. A working implementation may still be declined if its benefit does not justify its complexity or maintenance cost.

<a id="pull-requests"></a>

## Submitting and review

- Check existing [open and closed PRs](https://github.com/navidrome/navidrome/pulls?q=is%3Apr) to avoid duplicate work.
- Use the [development environment instructions](https://www.navidrome.org/docs/developers/dev-environment/). Work on a branch in your fork.
- Complete the [PR template](.github/pull_request_template.md) with the scope agreement or applicable exception, verification evidence, and maintenance implications. Keep each PR focused.
- Follow the [commit conventions and DCO requirements](#commit-conventions-and-dco) below.

Review and merge are discretionary. Proposals may be declined, simplified, split, or deferred. PRs that do not fit Navidrome's direction, the agreed scope, or the maintainer's capacity to maintain them may be closed without detailed code review or prolonged discussion. PRs lacking evidence or left with unresolved questions may also be closed. Agreement to pursue work does not guarantee review or merge. Please do not tag the maintainer, send private messages, or post status requests to obtain faster review.

## Commit conventions and DCO

Use this commit format: `<type>(scope): <description>`, with an optional body.

Allowed types: `feat`, `fix`, `sec`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `revert`, `chore`.

Use a relevant file or folder as the scope and a concise description. Reference the related issue when one exists, and include `BREAKING CHANGE` in the body for breaking changes. Sign off every commit using `git commit --signoff`. These requirements apply equally to AI-assisted work.
