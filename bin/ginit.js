#!/usr/bin/env node
const program = require('commander');
const packageJson = require('../package.json');

program
  .version(packageJson.version)
  .command('github', 'Initialize repo for GitHub')
  .command('gitlab', 'Initialize repo for Gitlab')
  .command('azure', 'Initialize repo for Azure DevOps')
  .command('bitbucket', 'Initialize repo for Bitbucket')
  .command('config', 'Manage configurations')
  .parse(process.argv);
