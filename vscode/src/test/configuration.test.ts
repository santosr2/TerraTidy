import * as assert from 'node:assert';
import { getInitializationOptions } from '../extension';

suite('Configuration', () => {
    test('getInitializationOptions returns expected shape', () => {
        const options = getInitializationOptions();

        assert.ok(typeof options === 'object', 'Should return an object');
        assert.ok('engines' in options, 'Should have engines');
        assert.ok('formatOnSave' in options, 'Should have formatOnSave');
        assert.ok('runOnSave' in options, 'Should have runOnSave');
        assert.ok('fixOnSave' in options, 'Should have fixOnSave');
    });

    test('engines have correct defaults', () => {
        const options = getInitializationOptions();
        const engines = options.engines as Record<string, boolean>;

        assert.strictEqual(engines.fmt, true, 'fmt should default to true');
        assert.strictEqual(engines.style, true, 'style should default to true');
        assert.strictEqual(engines.lint, true, 'lint should default to true');
        assert.strictEqual(engines.policy, false, 'policy should default to false');
    });

    test('severityThreshold is undefined when not explicitly set', () => {
        // severityThreshold is only sent when explicitly configured by the user
        // so that .terratidy.yaml config file values are respected
        const options = getInitializationOptions();
        assert.strictEqual(options.severityThreshold, undefined);
    });

    test('save options default to false', () => {
        const options = getInitializationOptions();
        assert.strictEqual(options.formatOnSave, false);
        assert.strictEqual(options.runOnSave, false);
        assert.strictEqual(options.fixOnSave, false);
    });

    /**
     * This test documents the mapping between VSCode settings and LSP InitializationOptions.
     * The mapping must match what the LSP server expects in internal/lsp/types.go.
     *
     * VSCode settings (package.json) -> LSP InitializationOptions:
     *   terratidy.profile           -> profile (string)
     *   terratidy.configPath        -> configPath (string)
     *   terratidy.engines.fmt       -> engines.fmt (boolean)
     *   terratidy.engines.style     -> engines.style (boolean)
     *   terratidy.engines.lint      -> engines.lint (boolean)
     *   terratidy.engines.policy    -> engines.policy (boolean)
     *   terratidy.severityThreshold -> severityThreshold (string, only if explicitly set)
     *   terratidy.formatOnSave      -> formatOnSave (boolean)
     *   terratidy.runOnSave         -> runOnSave (boolean)
     *   terratidy.fixOnSave         -> fixOnSave (boolean)
     *
     * Settings NOT sent to LSP:
     *   terratidy.executablePath    -> used by extension to locate binary
     *   terratidy.trace.server      -> standard LSP tracing, not custom option
     */
    test('all expected LSP options are present', () => {
        const options = getInitializationOptions();

        // These keys must match the JSON field names in internal/lsp/types.go InitializationOptions
        const expectedKeys = [
            'profile',
            'configPath',
            'engines',
            'severityThreshold',
            'formatOnSave',
            'runOnSave',
            'fixOnSave',
        ];

        for (const key of expectedKeys) {
            assert.ok(key in options, `Missing LSP option: ${key}`);
        }

        // Verify engines sub-structure
        const engines = options.engines as Record<string, boolean>;
        const engineKeys = ['fmt', 'style', 'lint', 'policy'];
        for (const key of engineKeys) {
            assert.ok(key in engines, `Missing engine toggle: ${key}`);
            assert.strictEqual(typeof engines[key], 'boolean', `Engine ${key} should be boolean`);
        }
    });

    test('optional string settings are undefined or string', () => {
        const options = getInitializationOptions();

        // profile and configPath are undefined when not set (empty string becomes undefined)
        const profile = options.profile;
        const configPath = options.configPath;

        assert.ok(profile === undefined || typeof profile === 'string', 'profile should be undefined or string');
        assert.ok(
            configPath === undefined || typeof configPath === 'string',
            'configPath should be undefined or string'
        );
    });
});
