# ginit

Initialize GIT repository in a folder.

This application automates the manual steps most of people do when creating new repositories

- initialize the local repository by running `git init`
- create a remote repository, for example on Github, Gitlab or Bitbucket - typically by leaving the command line and firing up a web browser
- add the remote
- create a `.gitignore` file (either using something like [https://gitignore.io](https://gitignore.io) or by copying their very own custom one)
- append to `.gitignore` files already in folder (you can choose which ones)
- create some config files (e.g. `.eslintrc`, `.tslintrc`, `.prettierrc`)
- add your project files
- commit the initial set of files
- push up to the remote repository

## Instalation

```sh
npm install -g https://github.com/stefanjarina/ginit
# -or-
yarn global add https://github.com/stefanjarina/ginit
```

## Usage

- From folder where you want to create a repo run:

```sh
ginit github
# -or-
ginit github --repo_name myrepo -description "This is my awesome repo"
```

## TODO

- [ ] Search for better name as ginit is already used on NPM
- Implement other repositories
  - [ ] Gitlab
  - [ ] BitBucket
  - [ ] Azure Devops
- [ ] Add other config files with sane defaults
- [ ] Add build system and Rewrite to TypeScript
- Quality control
  - [ ] Check if all errors are properly handled
  - [ ] Check if output messages make sense
- [ ] Add tests
- [ ] Publish to NPM
- And more ideas???
  - [ ] add config functionality for setting "presets" or defaults
  - [x] check if 'search' is possible in user inputs (implemented with inquirer checkbox-plus)
  - [ ] add dynamic config files creation (maybe place configs in some folder and they will get picked up by application)

## Commands
