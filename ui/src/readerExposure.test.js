import assert from 'node:assert/strict';
import test from 'node:test';
import {
  baseRoute,
  canonicalMCPName,
  componentSettingsOperation,
  derivedExposureState,
  exposureSetting,
  suggestExposureName,
  toolExposure,
  validateExposure,
} from './readerExposure.js';

const structure = {
  component: {
    name: 'Vendor Product Summary',
    routes: [{
      method: 'GET',
      path: '/v1/readers/vendor-summary',
      mcp: [
        { kind: 'resource', name: 'vendor.schema' },
        { kind: 'tool', name: 'vendor.summary.read', description: "Reader's summary" },
      ],
    }],
    settings: { report: { enabled: true, compose: { enabled: true, mcpTool: true } } },
  },
};

test('base MCP exposure selects the tool rather than an unrelated route resource', () => {
  assert.equal(baseRoute(structure).path, '/v1/readers/vendor-summary');
  assert.equal(toolExposure(structure).name, 'vendor.summary.read');
  assert.equal(suggestExposureName(structure.component, 'alice'), 'alice.vendor.summary.read');
  assert.equal(suggestExposureName({ name: 'reader', routes: [{ path: '/forecasting' }] }, 'alice'), 'alice.forecasting.read');
});

test('MCP setting quotes ordinary descriptions without browser-authored DQL', () => {
  assert.deepEqual(exposureSetting({
    enabled: true,
    name: 'vendor.summary.read',
    description: "Read vendor's grouped summary",
    descriptionPath: 'docs/vendor summary.md',
  }), {
    name: 'mcp',
    args: ['"vendor.summary.read"', '"Read vendor\'s grouped summary"', '"docs/vendor summary.md"'],
  });
  assert.deepEqual(exposureSetting({ enabled: false }), { name: 'mcp', remove: true });
});

test('MCP exposure validation requires a route, a stable name, and safe resource path', () => {
  const valid = { enabled: true, name: 'alice.vendor.summary.read', ownerPackage: 'alice', description: '', descriptionPath: 'docs/vendor.md', route: baseRoute(structure) };
  assert.equal(validateExposure(valid), '');
  assert.match(validateExposure({ ...valid, route: null }), /compiled HTTP route/);
  assert.match(validateExposure({ ...valid, name: 'vendor summary' }), /Tool name/);
  assert.match(validateExposure({ ...valid, name: '1alice.vendor.read' }), /canonical/);
  assert.equal(canonicalMCPName(' Alice Vendor / Read '), 'Alice.Vendor.Read');
  assert.match(validateExposure({ ...valid, name: 'bob.vendor.read' }), /owner prefix/);
  assert.match(validateExposure({ ...valid, descriptionPath: '../secret.md' }), /relative resource path/);
});

test('derived MCP status follows Datly report metadata', () => {
  assert.deepEqual(derivedExposureState(structure), { cube: true, compose: true });
  assert.deepEqual(derivedExposureState({ component: {} }), { cube: false, compose: false });
});

test('component settings batch cube, composition, and MCP atomically', () => {
  const operation = componentSettingsOperation({
    connector: 'main',
    cubeEnabled: true,
    composeEnabled: true,
    composeMCP: false,
    maxCubes: 4,
    maxLimit: 50,
    timeoutMs: 12000,
    exposure: { enabled: true, name: 'alice.vendor.read', description: 'Read vendors', descriptionPath: '' },
  });
  assert.equal(operation.type, 'batch');
  assert.deepEqual(operation.operations.map((item) => item.setting.name), ['connector', 'cube', 'cubeCompose', 'mcp', 'mcpOnly']);
  assert.deepEqual(operation.operations[0].setting.args, ['"main"']);
  assert.deepEqual(operation.operations[2].setting.args, ['true', 'false', '4', '50', '12000']);
  assert.equal(operation.operations[3].setting.args[0], '"alice.vendor.read"');
  assert.equal(operation.operations[4].setting.remove, true);

  const disabled = componentSettingsOperation({ connector: 'main', cubeEnabled: false, composeEnabled: false, exposure: { enabled: false } });
  assert.deepEqual(disabled.operations.map((item) => [item.setting.name, item.setting.remove]), [['connector', undefined], ['cubeCompose', true], ['cube', true], ['mcp', true], ['mcpOnly', true]]);

  const cubeWithoutMCP = componentSettingsOperation({ connector: 'main', cubeEnabled: true, cubeMCP: false, composeEnabled: false, exposure: { enabled: false } });
  assert.deepEqual(cubeWithoutMCP.operations[1].setting.args, ["''", "''", "''", "''", "''", "''", "''", 'false']);

  const pureMCP = componentSettingsOperation({ connector: 'main', cubeEnabled: false, composeEnabled: false, exposure: { enabled: true, mcpOnly: true, name: 'alice.reader', description: '', descriptionPath: '' } });
  assert.deepEqual(pureMCP.operations.at(-1).setting, { name: 'mcpOnly', args: ['true'] });
});
