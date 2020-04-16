const chalk = require('chalk');
const clear = require('clear');
const figlet = require('figlet');
const files = require('../lib/files');
const github = require('../lib/api/github');
const repo = require('../lib/repo');

const getGithubToken = async () => {
  // Fetch token from config store
  let token = github.getStoredGithubToken();
  if (token) {
    return token;
  }

  // No token found, use credentials to access GitHub account
  token = await github.getPersonalAccessToken();

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
      // Retrieve and Set Authentication Token
      const token = await getGithubToken();
      github.githubAuth(token);

      // Create remote repository
      const url = await repo.createRemoteRepo(repoName, description);

      // Create gitignore file
      await repo.createGitIgnoreFromApi();
      await repo.createGitIgnoreFromFiles();

      // Set up local repository and push to remote
      await repo.setupRepo(url);

      console.log(chalk.green('All done!'));
    } catch (err) {
      if (err) {
        switch (err.status) {
          case 401:
            console.error(
              chalk.red(
                "Couldn't log you in. Please provide correct credentials/token."
              )
            );
            break;
          case 422:
            console.log(
              chalk.red(
                'There is already a remote repository or token with the same name'
              )
            );
            break;
          default:
            console.log(chalk.red(err));
        }
      }
    }
  },
};
