const chalk = require('chalk');
const clear = require('clear');
const figlet = require('figlet');
const files = require('../lib/files');
const azure = require('../lib/api/azure');
const repo = require('../lib/repo');

const getAzureOrgNameAndToken = async () => {
  // Fetch token & orgUrl from config store
  const { token, orgName } = azure.getStoredOrgNameAndToken();
  if (token && orgName) {
    return { token, orgName };
  }

  // No token found, ask user for a token
  const authData = await azure.getOrgNameAndPersonalAccessToken();

  return authData;
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
      const { orgName, token } = await getAzureOrgNameAndToken();
      azure.authenticate(`https://dev.azure.com/${orgName}/`, token);

      // Create remote repository
      const url = await repo.createRemoteRepo('azure', repoName, description);

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
