# Navidrome Contribution Guide

Navidrome is a streaming service which allows you to enjoy your music collection from anywhere. We'd welcome you to contribute to our open source and make Navidrome even better. There are some basic guidelines which you need to follow if you like to contribute to Navidrome.

- [Code of Conduct](#code-of-conduct)
- [Issues](#issues)
- [Questions](#questions)
- [Pull Requests](#pull-requests)


## Code of Conduct
Please read the following [Code of Conduct](https://github.com/navidrome/navidrome/blob/master/CODE_OF_CONDUCT.md).

## Issues
Found any issue or bug in our codebase. You can help by submitting a issue to the Github repository. We would like issues created to have a following format:
`<Issue type>: <Issue Title>`

Issue type can be bug, feature or any other which is suitable for the issue.

The body of issue can be similar to:
```
    Issue Type

    Issue description
    
    Current Behaviour (If issue is bug)
    Expected Behaviour  (If issue is bug)

    Steps to reproduce

    Any other issues(that might be related)

    Context(Tell about the system used, browser, desktop version and any other details that might be needed)

    Do you want to work on the issue?
```
**Before opening a issue, please check that the issue is not opened earlier, as duplication of issues is not healthy**

## Questions
We would like to have discussions and general queries related to Navidrome on our [IRC channel](https://discord.gg/2qMuMyHfSV).

## Pull requests
Before submitting a pull request, ensure that you go through the following:
- Open a corresponding issue for the Pull Request, if not existing. The issue can be opened following [these guidelines](#issues)
- Ensure that there is no open or closed Pull Request corresponding to your submission to avoid duplication of effort.
- Setup the [development environment](https://www.navidrome.org/docs/developers/dev-environment/)
- Create a new branch on your forked repo and make the changes in it. Naming conventions for branch are: `<Issue Title>/<Issue Number>`. Example:
```
    git checkout -b adding-docs/I200 master
```
- The commits should follow a [specific convention](#commit-conventions)
- Ensure that a DCO sign-off for commits is provided via `--signoff` option of git commit
- Provide a link to the issue that will be closed via your Pull request.

### Commit Conventions
Each commit message must adhere to the following format:
```
<type>(scope): <description> - <issue number>

[optional body]
```
This improves the readability of the messages

#### Type
It can be one of the following:
1. **feat**: Addition of a new feature
2. **fix**: Bug fix
3. **docs**: Documentation Changes
4. **style**: Changes to styling
5. **refactor**: Refactoring of code
6. **perf**: Code that affects performance
7. **test**: Updating or improving the current tests
8. **build**: Changes to Build process
9. **revert**: Reverting to a previous commit 
10. **chore** : updating grunt tasks etc

If there is a breaking change in your Pull Request, please add `BREAKING CHANGE` in the optional body section

#### Scope
The file or folder where the changes are made. If there are more than one, you can mention any

#### Description
A short description of the issue

#### Issue number
The issue fixed by this Pull Request.

The body is optional. It may contain short description of changes made.

Following all the guidelines an ideal commit will look like:
```
    git commit --signoff -m "feat(themes): New-theme - I219"
```

After commiting push your commits to your forked branch and create a Pull Request from there.
The Pull Request Title can be the same as `<type>(scope): <description> - <issue number>`
A demo layout of how the Pull request body can look:
```
Closes <Issue number along with link>

Description (What does the pull request do)

Changes (What changes were made )

Screenshots or Videos

Related Issues and Pull Requests(if any)

```
