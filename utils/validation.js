module.exports = {
  isRequired: (input, field) =>
    input.trim().length ? true : `Please enter your ${field}.`,
  isRequiredForRepo: (input, field) =>
    input.trim().length ? true : `Please enter a ${field} for the repository.`,
};
