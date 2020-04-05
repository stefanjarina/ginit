const { Spinner } = require('clui');
const fs = require('fs');
const git = require('simple-git/promise')();
const touch = require('touch');
const _ = require('lodash');
const gitignore = require('gitignore.io');

const prompts = require('./prompts');
const gh = require('./github');
const files = require('./files');

module.exports = {
  createRemoteRepo: async (repoName, description) => {
    const github = gh.getInstance();
    const answers = await prompts.askRepoDetails(repoName, description);

    const data = {
      name: answers.name,
      description: answers.description,
      private: answers.visibility === 'private',
    };

    const status = new Spinner('Creating remote repository...');
    status.start();

    try {
      const response = await github.repos.createForAuthenticatedUser(data);

      if (process.platform === 'win32') {
        return response.data.html_url;
      }
      return response.data.ssh_url;
    } finally {
      status.stop();
    }
  },
  createGitIgnoreFromFiles: async () => {
    const filelist = _.without(fs.readdirSync('.'), '.git', '.gitignore');

    if (filelist.length) {
      const answers = await prompts.askGitIgnore(filelist);

      if (answers.ignore.length) {
        if (!files.directoryExists('.gitignore')) {
          fs.appendFileSync('.gitignore', answers.ignore.join('\n'));
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
      if (!files.directoryExists('.gitignore')) {
        const ignoreString = await gitignore.fetchConfig([...answers.ignore]);

        fs.appendFileSync('.gitignore', ignoreString);
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
