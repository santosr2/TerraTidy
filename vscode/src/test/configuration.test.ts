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
});
