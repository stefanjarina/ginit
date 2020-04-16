const chalk = require('chalk');
const clear = require('clear');
const figlet = require('figlet');
const files = require('../lib/files');
const gitlab = require('../lib/api/gitlab');
const repo = require('../lib/repo');

const getGitlabToken = async () => {
  // Fetch token from config store
  let token = gitlab.getStoredToken();
  if (token) {
    return token;
  }

  // No token found, use credentials to access GitHub account
  token = await gitlab.getPersonalAccessToken();

  return token;
};

module.exports = {
  run: async (repoName, description) => {
    clear();

    console.log(
      chalk.yellow(figlet.textSync('Ginit', { horizontalLayout: 'full' }))
    );

    if (files.directoryExists('.git')) {
      console.log(chalk.red('Already a Git repository!'));
      process.exit();
    }

    try {
      // Retrieve and set authentication token
      const token = await getGitlabToken();
      gitlab.authenticate(token);

      // Create remote repository
      const url = await repo.createRemoteRepo('gitlab', repoName, description);

      // Create gitignore file
      await repo.createGitIgnoreFromApi();
      await repo.createGitIgnoreFromFiles();

      // Set up local repository and push to remote
      await repo.setupRepo(url);

      console.log(chalk.green('All done!'));
    } catch (err) {
      console.log(chalk.red(err));
    }
  },
};
