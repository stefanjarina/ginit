const { Spinner } = require('clui');
const fs = require('fs');
const git = require('simple-git/promise')();
const touch = require('touch');
const _ = require('lodash');
const gitignore = require('gitignore.io');

const prompts = require('./prompts');
const gh = require('./api/github');
const gitlab = require('./api/gitlab');
const files = require('./files');

module.exports = {
  createRemoteRepo: async (repository, repoName, description) => {
    let url;

    switch (repository) {
      case 'github':
        url = await gh.createRepository(repoName, description);
        break;
      case 'gitlab':
        url = await gitlab.createRepository(repoName, description);
        break;
      case 'azure':
        url = await gh.createRepository(repoName, description);
        break;
      case 'bitbucket':
        url = await gh.createRepository(repoName, description);
        break;
      default:
        throw Error('This repository is not supported');
    }

    return url;
  },
  createGitIgnoreFromFiles: async () => {
    const filelist = _.without(fs.readdirSync('.'), '.git', '.gitignore');

    if (filelist.length) {
      const answers = await prompts.askGitIgnore(filelist);

      if (answers.ignore.length) {
        if (files.directoryExists('.gitignore')) {
          fs.appendFileSync('.gitignore', answers.ignore.join('\n'));
        } else {
          fs.writeFileSync('.gitignore', answers.ignore.join('\n'));
        }
      } else {
        touch('.gitignore');
      }
    } else {
      touch('.gitignore');
    }
  },
  createGitIgnoreFromApi: async () => {
    const answers = await prompts.askGitIgnore();

    if (answers.ignore.length) {
      const ignoreString = await gitignore.fetchConfig([...answers.ignore]);
      if (files.directoryExists('.gitignore')) {
        fs.appendFileSync('.gitignore', ignoreString);
      } else {
        fs.writeFileSync('.gitignore', ignoreString);
      }
    } else {
      touch('.gitignore');
    }
  },
  setupRepo: async url => {
    const status = new Spinner(
      'Initializing local repository and pushing to remote...'
    );
    status.start();

    try {
      git
        .init()
        .then(git.add('.gitignore'))
        .then(git.add('./*'))
        .then(git.commit('initial commit'))
        .then(git.addRemote('origin', url))
        .then(git.push('origin', 'master'));
    } finally {
      status.stop();
    }
  },
};
