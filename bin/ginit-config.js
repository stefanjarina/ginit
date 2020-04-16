const program = require('commander');
const configCmd = require('../commands/config');

program
  .name('ginit config')
  .requiredOption(
    '-r, --repo <github|devops|gitlab|bitbucket>',
    'specify repository'
  );

program
  .command('get')
  .description('set value of a configuration key')
  .arguments('<key>')
  .action((key, cmd) => {
    configCmd.get(cmd.parent.repo, key);
  });

program
  .command('all')
  .description('lists whole configuration')
  .action(cmd => {
    configCmd.all(cmd.parent.repo);
  });

program
  .command('set')
  .description('set configuration key to a specified value')
  .arguments('<key> <value>')
  .action((key, value, cmd) => {
    configCmd.set(cmd.parent.repo, key, value);
  });

program
  .command('remove')
  .description('remove a configuration key')
  .arguments('<key>')
  .action((key, cmd) => {
    configCmd.remove(cmd.parent.repo, key);
  });

program.parse(process.argv);
