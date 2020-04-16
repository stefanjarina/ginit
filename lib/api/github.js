const { Spinner } = require('clui');
const { Octokit } = require('@octokit/rest');
const { createBasicAuth } = require('@octokit/auth-basic');
const ConfigManager = require('../ConfigManager');
const prompts = require('../prompts');

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
    const configManager = new ConfigManager('github');
    return configManager.get('token');
  },

  getPersonalAccessToken: async () => {
    let token = null;
    const authMethod = await prompts.askGithubAuthenticationMethod();
    if (authMethod.authenticationMethod === 'token') {
      token = prompts.askForToken();
      return token;
    }
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
        const configManager = new ConfigManager('github');
        configManager.set('token', res.token);
        return res.token;
      }

      throw new Error('GitHub token was not found in the response');
    } finally {
      status.stop();
    }
  },
};
