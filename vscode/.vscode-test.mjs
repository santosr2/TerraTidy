import { defineConfig } from '@vscode/test-cli';

export default defineConfig({
    files: 'out/test/**/*.test.js',
    workspaceFolder: 'src/test/fixtures',
    mocha: {
        timeout: 20_000,
    },
});
