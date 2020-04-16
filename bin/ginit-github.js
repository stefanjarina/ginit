const program = require('commander');
const githubCmd = require('../commands/github');

program
  .name('ginit github')
  .arguments('[repo_name] [description]')
  .action((repoName, description) => {
    githubCmd.run(repoName, description);
  });

program.parse(process.argv);
