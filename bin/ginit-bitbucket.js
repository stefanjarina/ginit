const program = require('commander');
const bitbucketCmd = require('../commands/bitbucket');

program
  .name('ginit bitbucket')
  .arguments('[repo_name] [description]')
  .action((repoName, description) => bitbucketCmd.run(repoName, description));

program.parse(process.argv);
