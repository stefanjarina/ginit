const { Spinner } = require('clui');
const Configstore = require('configstore');
const { Octokit } = require('@octokit/rest');
const { createBasicAuth } = require('@octokit/auth-basic');
const prompts = require('./prompts');
const packageJson = require('../package.json');

const conf = new Configstore(packageJson.name);

let octokit;

module.exports = {
  githubAuth: token => {
    octokit = new Octokit({
      auth: token,
    });
  },

  getInstance: () => {
    return octokit;
  },

  getStoredGithubToken: () => {
    return conf.get('github.token');
  },

  getPersonalAccessToken: async () => {
    const credentials = await prompts.askGithubCredentials();
    const status = new Spinner('Authentication in progress, please wait...');

    status.start();

    const auth = createBasicAuth({
      username: credentials.username,
      password: credentials.password,
      async on2Fa() {
        status.stop();
        const res = await prompts.getTwoFactorAuthenticationCode();
        status.start();
        return res.twoFactorAuthenticationCode;
      },
      token: {
        scopes: ['user', 'public_repo', 'repo', 'repo:status'],
        note: 'ginit, the command-line tool for initializing Git repos',
      },
    });

    try {
      const res = await auth();

      if (res.token) {
        conf.set('github.token', res.token);
        return res.token;
      }

      throw new Error('GitHub token was not found in the response');
    } finally {
      status.stop();
    }
  },
};
