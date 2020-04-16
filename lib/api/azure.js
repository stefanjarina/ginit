const { Spinner } = require('clui');
const {
  WebApi,
  getPersonalAccessTokenHandler,
} = require('azure-devops-node-api');
const ConfigManager = require('../ConfigManager');
const prompts = require('../prompts');

let devops;

module.exports = {
  authenticate: (orgUrl, token) => {
    const authHandler = getPersonalAccessTokenHandler(token);
    devops = new WebApi(orgUrl, authHandler);
  },

  getInstance: () => {
    return devops;
  },

  getStoredOrgNameAndToken: () => {
    const configManager = new ConfigManager('azure');
    const orgName = configManager.get('orgName');
    const token = configManager.get('token');
    return { orgName, token };
  },

  getOrgNameAndPersonalAccessToken: async () => {
    const { orgName, token } = await prompts.askForAzureAuth();
    const configManager = new ConfigManager('azure');
    configManager.set('orgName', orgName);
    configManager.set('token', token);
    return { orgName, token };
  },

  createRepository: async (repoName, desc) => {
    // Get Projects
    const projectStatus = new Spinner('Getting list of your projects...');
    projectStatus.start();
    let projects;
    try {
      const core = await devops.getCoreApi();
      projects = await core.getProjects();
    } catch (error) {
      throw new Error('Error getting required information from Azure DevOps');
    } finally {
      projectStatus.stop();
    }
    // get user answers
    const { name } = await prompts.askRepoDetails('azure', repoName, desc);
    const { projectId } = await prompts.askWhichAzureProject(projects);

    const data = {
      name,
      project: {
        id: projectId,
      },
    };

    const status = new Spinner('Creating remote repository...');
    status.start();

    try {
      const gitApi = await devops.getGitApi();
      const response = await gitApi.createRepository(data);
      if (process.platform === 'win32') {
        return response.remoteUrl;
      }
      return response.sshUrl;
    } finally {
      status.stop();
    }
  },
};
