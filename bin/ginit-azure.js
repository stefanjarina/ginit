const program = require('commander');
const azureCmd = require('../commands/azure');

program
  .name('ginit azure')
  .arguments('[repo_name] [description]')
  .action((repoName, description) => azureCmd.run(repoName, description));

program.parse(process.argv);
