const { Spinner } = require('clui');
const { Octokit } = require('@octokit/rest');
const { createBasicAuth } = require('@octokit/auth-basic');
const ConfigManager = require('../ConfigManager');
const prompts = require('../prompts');

let octokit;

module.exports = {
  authenticate: token => {
    octokit = new Octokit({
      auth: token,
    });
  },

  getInstance: () => {
    return octokit;
  },

  getStoredToken: () => {
    const configManager = new ConfigManager('github');
    return configManager.get('token');
  },

  getPersonalAccessToken: async () => {
    let token = null;
    const authMethod = await prompts.askGithubAuthenticationMethod();
    if (authMethod.authenticationMethod === 'token') {
      token = prompts.askForToken();
      const configManager = new ConfigManager('github');
      configManager.set('token', token);
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

  createRepository: async (repoName, description) => {
    const answers = await prompts.askRepoDetails(
      'github',
      repoName,
      description
    );

    const data = {
      name: answers.name,
      description: answers.description,
      private: answers.visibility === 'private',
    };

    const status = new Spinner('Creating remote repository...');
    status.start();

    try {
      const response = await octokit.repos.createForAuthenticatedUser(data);

      if (process.platform === 'win32') {
        return response.data.html_url;
      }
      return response.data.ssh_url;
    } finally {
      status.stop();
    }
  },
};
