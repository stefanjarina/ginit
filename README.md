# novugit

Initialize GIT repository in a folder.

This application automates the manual steps most of people do when creating new repositories

- initialize the local repository by running `git init`
- create a remote repository, for example on Github, Gitlab or Azure - typically by leaving the command line and firing up a web browser
- add the remote
- create a `.gitignore` file (either using something like [https://gitignore.io](https://gitignore.io) or by copying their very own custom one)
- append to `.gitignore` files already in folder (you can choose which ones)
- create some config files (e.g. `.eslintrc`, `.tslintrc`, `.prettierrc`) [not implemented yet]
- add your project files
- commit the initial set of files
- push up to the remote repository

## DISCLAIMER

THIS IS STILL IN PROVE OF CONCEPT AND BEING DEVELOPED

## Instalation

TBD once app is usable

## Usage

- From folder where you want to create a repo run:

```sh
ginit init github
# -or-
ginit init github --repo-name myrepo --description "This is my awesome repo"
```

## TODO

- Implement repositories
  - [ ] Gitlab
  - [ ] BitBucket (optional as I am not using it)
  - [ ] Azure Devops
- [ ] Add other config files with sane defaults
- [ ] Add build system (github actions?)
- Quality control
  - [ ] Check if all errors are properly handled
  - [ ] Check if output messages make sense
- [ ] Add tests
- [ ] Publish (scoop? chocolatey? nuget?)
- And more ideas???
  - [ ] add config functionality for setting "presets" or defaults
  - [ ] add dynamic config files creation (maybe place configs in some folder and they will get picked up by application)
  - [ ] config file encryption
