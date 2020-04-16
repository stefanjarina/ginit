const ConfigManager = require('../lib/ConfigManager');

module.exports = {
  set: (repository, key, value) => {
    const configManager = new ConfigManager(repository);
    configManager.set(key, value);
  },
  get: (repository, key) => {
    const configManager = new ConfigManager(repository);
    console.log(configManager.get(key));
  },
  all: repository => {
    const configManager = new ConfigManager(repository);
    console.log(configManager.all());
  },
  remove: (repository, key) => {
    const configManager = new ConfigManager(repository);
    configManager.remove(key);
  },
};
