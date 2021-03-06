const inquirer = require('inquirer');
const gitignore = require('gitignore.io');
const fuzzy = require('fuzzy');

const { isRequired, isRequiredForRepo } = require('../utils/validation');
const files = require('./files');

inquirer.registerPrompt(
  'checkbox-plus',
  require('inquirer-checkbox-plus-prompt')
);

module.exports = {
  askGithubAuthenticationMethod: () => {
    return inquirer.prompt({
      type: 'list',
      name: 'authenticationMethod',
      message: 'Authentification method:',
      choices: [
        {
          name: 'Personal Access Token',
          value: 'token',
        },
        {
          name: 'Username & Password',
          value: 'username_password',
        },
      ],
      default: 'token',
    });
  },

  askWhichGitlabGroup: (user, userNamespaceId, groups) => {
    const choices = [
      { name: user.name, value: userNamespaceId },
      ...groups.map(group => ({
        name: group.full_name,
        value: group.id,
      })),
    ];
    return inquirer.prompt({
      type: 'list',
      name: 'group',
      message: 'Which group to use?',
      choices,
    });
  },

  askWhichAzureProject: projects => {
    const choices = projects.map(project => ({
      name: project.name,
      value: project.id,
    }));
    return inquirer.prompt({
      type: 'list',
      name: 'projectId',
      message: 'Which project to use?',
      choices,
    });
  },

  askForAzureAuth: () => {
    const questions = [
      {
        type: 'input',
        name: 'orgName',
        message:
          'Enter your organization name (https://dev.azure.com/{yourorgname})',
        validate: value => isRequiredForRepo(value, 'name'),
      },
      {
        type: 'input',
        name: 'token',
        message: 'Enter your Personal Access Token',
        validate: value => isRequiredForRepo(value, 'name'),
      },
    ];
    return inquirer.prompt(questions);
  },

  askForToken: () => {
    return inquirer.prompt({
      type: 'input',
      name: 'token',
      message: 'Enter your Personal Access Token',
      validate: value => isRequiredForRepo(value, 'name'),
    });
  },

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

  askRepoDetails: (repository, repoName, description) => {
    let visibilityChoices;

    switch (repository) {
      case 'github':
        visibilityChoices = ['private', 'public'];
        break;
      case 'gitlab':
        visibilityChoices = ['private', 'internal', 'public'];
        break;
      case 'azure':
        visibilityChoices = null;
        break;
      case 'bitbucket':
        visibilityChoices = ['private', 'public'];
        break;
      default:
        throw Error('This repository is not supported');
    }

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
    ];

    if (visibilityChoices) {
      questions.push({
        type: 'list',
        name: 'visibility',
        message: 'Visibility of a repository:',
        choices: visibilityChoices,
        default: 'public',
      });
    }

    return inquirer.prompt(questions);
  },

  askGitIgnore: async filelist => {
    let questions = [];
    const namesList = await gitignore.listAll();
    if (filelist) {
      questions = [
        {
          type: 'checkbox-plus',
          name: 'ignore',
          highlight: true,
          searchable: true,
          message: 'Select the files and/or folders you wish to ignore:',
          default: ['node_modules'],
          source: (answersSoFar, input) => {
            const filter = input || '';

            return new Promise(resolve => {
              const fuzzyResult = fuzzy.filter(filter, filelist);

              const data = fuzzyResult.map(element => {
                return element.original;
              });

              resolve(data);
            });
          },
        },
      ];
    } else {
      questions = [
        {
          type: 'checkbox-plus',
          name: 'ignore',
          message:
            'Select config names you wish to fetch from https://gitignore.io',
          highlight: true,
          searchable: true,
          default: [
            'node',
            'visualstudiocode',
            'windows',
            'linux',
            'macos',
            'webstorm+all',
          ],
          source: (answersSoFar, input) => {
            const filter = input || '';

            return new Promise(resolve => {
              const fuzzyResult = fuzzy.filter(filter, namesList);

              const data = fuzzyResult.map(element => {
                return element.original;
              });

              resolve(data);
            });
          },
        },
      ];
    }
    return inquirer.prompt(questions);
  },
};
