const program = require('commander');

program
  .name('ginit github')
  .option('--test', 'test option', false)
  .action(cmd => console.log(`${cmd}`));

program.parse(process.argv);
