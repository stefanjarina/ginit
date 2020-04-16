const program = require('commander');
const gitlabCmd = require('../commands/gitlab');

program
  .name('ginit gitlab')
  .arguments('[repo_name] [description]')
  .action((repoName, description) => gitlabCmd.run(repoName, description));

program.parse(process.argv);
