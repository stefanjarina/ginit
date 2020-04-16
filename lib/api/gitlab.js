const { Spinner } = require('clui');
const { Gitlab } = require('@gitbeaker/node');
const ConfigManager = require('../ConfigManager');
const prompts = require('../prompts');

let gitlab;

module.exports = {
  authenticate: token => {
    gitlab = new Gitlab({
      token,
    });
  },

  getInstance: () => {
    return gitlab;
  },

  getStoredToken: () => {
    const configManager = new ConfigManager('gitlab');
    return configManager.get('token');
  },

  getPersonalAccessToken: async () => {
    const { token } = await prompts.askForToken();
    const configManager = new ConfigManager('gitlab');
    configManager.set('token', token);
    return token;
  },

  createRepository: async (repoName, desc) => {
    // Get groups
    const groupsStatus = new Spinner('Getting list of your groups...');
    groupsStatus.start();
    let user;
    let groups;
    let userNamespaceId;
    try {
      user = await gitlab.Users.current();
      const namespace = await gitlab.Namespaces.all({ search: user.name });
      userNamespaceId = namespace.id;
      groups = await gitlab.Groups.all();
    } catch (error) {
      throw new Error('Error getting required information from Gitlab');
    } finally {
      groupsStatus.stop();
    }
    // get user answers
    const { name, description, visibility } = await prompts.askRepoDetails(
      'gitlab',
      repoName,
      desc
    );
    const { group: namespaceId } = await prompts.askWhichGitlabGroup(
      user,
      userNamespaceId,
      groups
    );

    const data = {
      name,
      description,
      namespace_id: namespaceId,
      visibility,
    };

    const status = new Spinner('Creating remote repository...');
    status.start();

    try {
      const response = await gitlab.Projects.create(data);

      if (process.platform === 'win32') {
        return response.http_url_to_repo;
      }
      return response.ssh_url_to_repo;
    } finally {
      status.stop();
    }
  },
};
