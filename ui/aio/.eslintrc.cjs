module.exports = {
  root: true,
  extends: ['plugin:vue/vue3-recommended', require.resolve('@ainou/code-style')],
  parser: 'vue-eslint-parser',
  parserOptions: {
    parser: '@typescript-eslint/parser',
    // tsconfigRootDir: __dirname,
    // project: ['./tsconfig.eslint.json'],
  },
  rules: {
    'vue/html-self-closing': 'off',
    'vue/max-attributes-per-line': 'off',
    'vue/singleline-html-element-content-newline': 'off',
  },
}
