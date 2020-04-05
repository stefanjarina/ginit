const inquirer = require('inquirer');
const gitignore = require('gitignore.io');
const { isRequired, isRequiredForRepo } = require('../utils/validation');
const files = require('./files');

module.exports = {
  askGithubCredentials: () => {
    const questions = [
      {
        name: 'username',
        type: 'input',
        message: 'Enter your GitHub username or email address:',
        validate: value => isRequired(value, 'username or email address'),
      },
      {
        name: 'password',
        type: 'password',
        message: 'Enter your password:',
        validate: value => isRequired(value, 'password'),
      },
    ];
    return inquirer.prompt(questions);
  },

  getTwoFactorAuthenticationCode: () => {
    return inquirer.prompt({
      name: 'twoFactorAuthenticationCode',
      type: 'input',
      message: 'Enter your two-factor authentication code:',
      validate: value => isRequired(value, 'two-factor authentication code'),
    });
  },

  askRepoDetails: (repoName, description) => {
    const questions = [
      {
        type: 'input',
        name: 'name',
        message: 'Enter a name for the repository',
        default: repoName || files.getCurrentDirectoryBase(),
        validate: value => isRequiredForRepo(value, 'name'),
      },
      {
        type: 'input',
        name: 'description',
        default: description || null,
        message: 'Optionally enter a description of the repository:',
      },
      {
        type: 'list',
        name: 'visibility',
        message: 'Public or private:',
        choices: ['public', 'private'],
        default: 'public',
      },
    ];
    return inquirer.prompt(questions);
  },

  askGitIgnore: async filelist => {
    let questions = [];
    if (filelist) {
      questions = [
        {
          type: 'checkbox',
          name: 'ignore',
          message: 'Select the files and/or folders you wish to ignore:',
          choices: filelist,
          default: ['node_modules', 'bower_components'],
        },
      ];
    } else {
      questions = [
        {
          type: 'checkbox',
          name: 'ignore',
          message:
            'Select config names you wish to fetch from https://gitignore.io',
          choices: await gitignore.listAll(),
          default: [
            'node',
            'visualstudiocode',
            'windows',
            'linux',
            'macos',
            'webstorm+all',
          ],
        },
      ];
    }
    return inquirer.prompt(questions);
  },
};
