#!/usr/bin/env node
const program = require('commander');
const packageJson = require('../package.json');

program
  .version(packageJson.version)
  .command('github', 'Initialize repo for GitHub')
  .command('config', 'Manage configurations')
  .parse(process.argv);
