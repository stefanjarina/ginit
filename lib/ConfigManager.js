const Configstore = require('configstore');
const packageJson = require('../package.json');

const supportedRepositories = ['github', 'gitlab', 'azure', 'bitbucket'];

class ConfigManager {
  constructor(repo) {
    this.conf = new Configstore(packageJson.name);
    this.repo = ConfigManager.validateRepo(repo);
  }

  set(key, value) {
    this.conf.set(`${this.repo}.${key}`, value);
  }

  get(key) {
    return this.conf.get(`${this.repo}.${key}`);
  }

  all() {
    return this.conf.all[this.repo];
  }

  remove(key) {
    this.conf.delete(`${this.repo}.${key}`);
  }

  static validateRepo(repo) {
    if (supportedRepositories.includes(repo)) {
      return repo;
    }
    throw Error('This repository is not supported yet.');
  }
}

module.exports = ConfigManager;
