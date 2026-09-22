import React, { useEffect, useState } from 'react';
import {
  Button,
  Callout,
  Code,
  Dialog,
  DialogBody,
  DialogFooter,
  FormGroup,
  HTMLSelect,
  Icon,
  InputGroup,
  Spinner,
  Switch,
  Tag,
  TextArea,
} from '@blueprintjs/core';
import {
  baseRoute,
  componentSettingsOperation,
  canonicalMCPName,
  suggestExposureName,
  toolExposure,
  validateExposure,
} from './readerExposure.js';

export function ReaderExposureDialog({ isOpen, api, structure, version, report, onClose, onApply, onReportUpdated }) {
  const component = structure?.component;
  const route = baseRoute(structure);
  const exposure = toolExposure(structure);
  const reportSettings = component?.settings?.report;
  const compose = reportSettings?.compose;
  const [cubeEnabled, setCubeEnabled] = useState(Boolean(reportSettings?.enabled));
  const [cubeMCP, setCubeMCP] = useState(reportSettings?.mcpTool == null ? true : Boolean(reportSettings.mcpTool));
  const [composeEnabled, setComposeEnabled] = useState(Boolean(compose?.enabled));
  const [composeMCP, setComposeMCP] = useState(Boolean(compose?.mcpTool));
  const [maxCubes, setMaxCubes] = useState(String(compose?.maxCubes ?? 8));
  const [maxLimit, setMaxLimit] = useState(String(compose?.maxLimit ?? 100));
  const [timeout, setTimeoutMs] = useState(String(compose?.timeoutMs ?? 30000));
  const [enabled, setEnabled] = useState(Boolean(exposure));
  const [mcpOnly, setMCPOnly] = useState(Boolean(route?.internal));
  const [name, setName] = useState(exposure?.name ?? '');
  const [description, setDescription] = useState(exposure?.description ?? '');
  const [descriptionPath, setDescriptionPath] = useState(exposure?.descriptionPath ?? '');
  const [title, setTitle] = useState(report?.title ?? '');
  const [componentDescription, setComponentDescription] = useState(report?.description ?? '');
  const [connector, setConnector] = useState(report?.defaultConnectorName ?? '');
  const [connectors, setConnectors] = useState([]);
  const [loadingConnectors, setLoadingConnectors] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!isOpen) return;
    setCubeEnabled(Boolean(reportSettings?.enabled));
    setCubeMCP(reportSettings?.mcpTool == null ? true : Boolean(reportSettings.mcpTool));
    setComposeEnabled(Boolean(compose?.enabled));
    setComposeMCP(Boolean(compose?.mcpTool));
    setMaxCubes(String(compose?.maxCubes ?? 8));
    setMaxLimit(String(compose?.maxLimit ?? 100));
    setTimeoutMs(String(compose?.timeoutMs ?? 30000));
    setEnabled(Boolean(exposure));
    setMCPOnly(Boolean(route?.internal));
    setName(exposure?.name ?? suggestExposureName(component, report?.ownerPackage));
    setDescription(exposure?.description ?? component?.description ?? '');
    setDescriptionPath(exposure?.descriptionPath ?? '');
    setTitle(report?.title ?? '');
    setComponentDescription(report?.description ?? '');
    setConnector(report?.defaultConnectorName ?? '');
    setSaving(false);
    setError('');
    if (api) {
      setLoadingConnectors(true);
      api.listConnectors({ status: 'active', limit: 100 }).then((page)=>setConnectors(page?.items??[])).catch((cause)=>setError(cause.message)).finally(()=>setLoadingConnectors(false));
    }
  }, [api, isOpen, exposure?.name, exposure?.description, exposure?.descriptionPath, route?.internal, component?.name, report?.ownerPackage, report?.title, report?.description, report?.defaultConnectorName, reportSettings?.enabled, reportSettings?.mcpTool, compose?.enabled, compose?.mcpTool, compose?.maxCubes, compose?.maxLimit, compose?.timeoutMs]);

  const submit = async (event) => {
    event.preventDefault();
    const message = validateExposure({ enabled, name, description, descriptionPath, route, ownerPackage: report?.ownerPackage });
    if (message) {
      setError(message);
      return;
    }
    if (!title.trim() || !connector) {
      setError('Enter a component title and select an active connector.');
      return;
    }
    const budgets = [maxCubes, maxLimit, timeout].map(Number);
    if (composeEnabled && budgets.some((value) => !Number.isInteger(value) || value <= 0)) {
      setError('Maximum cubes, result rows, and timeout must be positive integers.');
      return;
    }
    setSaving(true);
    setError('');
    try {
      await onApply(componentSettingsOperation({ connector, cubeEnabled, cubeMCP, composeEnabled, composeMCP, maxCubes: budgets[0], maxLimit: budgets[1], timeoutMs: budgets[2], exposure: { enabled, mcpOnly, name, description, descriptionPath } }));
      if (api && report?.id) {
        let current = await api.getReport(report.id);
        if (title.trim() !== current.title || componentDescription.trim() !== (current.description || '')) {
          current = await api.updateReport(report.id, { title: title.trim(), description: componentDescription.trim(), etag: current.etag });
        }
        onReportUpdated?.(current);
      }
      onClose();
    } catch (cause) {
      setError(cause.message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog
      className="studio-connector-dialog studio-exposure-dialog"
      isOpen={isOpen}
      onClose={onClose}
      title="Edit component"
      icon="cog"
      canOutsideClickClose={!saving}
    >
      <form onSubmit={submit}>
        <DialogBody className="studio-connector-dialog-body">
          <p className="studio-dialog-lead">Manage the component identity, connector, analytics behavior, and MCP exposure in one place. View SQL and column contracts remain scoped to their selected view.</p>
          {error && <Callout intent="danger" role="alert" className="studio-dialog-callout">{error}</Callout>}

          <section className="studio-component-settings-section" aria-labelledby="component-identity-title">
            <div><h3 id="component-identity-title">Component</h3><p>Catalog identity and the active connector used by this versioned reader.</p></div>
            <FormGroup label="Component name" labelFor="component-title" required><InputGroup id="component-title" value={title} onChange={(event)=>setTitle(event.target.value)} disabled={saving}/></FormGroup>
            <FormGroup label="Connector" labelFor="component-connector" helperText="Changing the connector updates the versioned Datly connector setting and the component catalog together." required>{loadingConnectors?<Spinner size={20}/>:<HTMLSelect id="component-connector" value={connector} onChange={(event)=>setConnector(event.target.value)} fill disabled={saving}><option value="">Select an active connector</option>{connectors.map((item)=><option key={item.name} value={item.name}>{item.name} · {item.driver}</option>)}</HTMLSelect>}</FormGroup>
            <FormGroup label="Description" labelFor="component-description"><TextArea id="component-description" value={componentDescription} onChange={(event)=>setComponentDescription(event.target.value)} fill rows={2} disabled={saving}/></FormGroup>
          </section>

          <section className="studio-exposure-contract" aria-label="Compiled route contract">
            <div>
              <Icon icon="link" />
              <span>HTTP route</span>
            </div>
            <Code>{route ? `${route.method} ${route.path}` : 'No compiled route'}</Code>
            <Tag minimal intent={version?.compileStatus === 'valid' ? 'success' : 'warning'}>
              {version?.compileStatus ?? 'draft'}
            </Tag>
          </section>

          <section className="studio-component-settings-section" aria-labelledby="analytics-settings-title">
            <div><h3 id="analytics-settings-title">Analytics</h3><p>Cube validation and bounded composition apply to the complete reader component.</p></div>
            <Switch checked={cubeEnabled} label="Enable cube" onChange={(event) => { const checked = event.target.checked; setCubeEnabled(checked); if (!checked) setComposeEnabled(false); }} disabled={saving}/>
            {cubeEnabled && <Switch checked={cubeMCP} label="Expose cube as an MCP tool" onChange={(event)=>setCubeMCP(event.target.checked)} disabled={saving}/>}
            <Switch checked={composeEnabled} label="Enable cube composition" onChange={(event) => setComposeEnabled(event.target.checked)} disabled={saving || !cubeEnabled}/>
            {composeEnabled && <div className="studio-component-settings-details"><div className="studio-form-grid"><FormGroup label="Maximum cubes" labelFor="compose-max-cubes"><InputGroup id="compose-max-cubes" value={maxCubes} onChange={(event) => setMaxCubes(event.target.value)} inputMode="numeric" disabled={saving}/></FormGroup><FormGroup label="Maximum result rows" labelFor="compose-max-limit"><InputGroup id="compose-max-limit" value={maxLimit} onChange={(event) => setMaxLimit(event.target.value)} inputMode="numeric" disabled={saving}/></FormGroup></div><FormGroup label="Timeout (ms)" labelFor="compose-timeout"><InputGroup id="compose-timeout" value={timeout} onChange={(event) => setTimeoutMs(event.target.value)} inputMode="numeric" disabled={saving}/></FormGroup><Switch checked={composeMCP} label="Expose composition as an MCP tool" onChange={(event) => setComposeMCP(event.target.checked)} disabled={saving}/></div>}
          </section>

          <section className="studio-component-settings-section" aria-labelledby="mcp-settings-title">
            <div><h3 id="mcp-settings-title">MCP reader tool</h3><p>The named tool reuses this route’s typed input, authorization predicates, output contract, and connector bindings.</p></div>

          <Switch
            checked={enabled}
            label="Enable MCP tool"
            onChange={(event) => setEnabled(event.target.checked)}
            disabled={saving || !route}
          />

          {enabled && <Switch checked={mcpOnly} label="Publish as MCP-only" onChange={(event) => setMCPOnly(event.target.checked)} disabled={saving}/>}
          {enabled && mcpOnly && <Callout compact intent="primary">The component route remains available to Datly’s runtime and MCP server, but is excluded from public HTTP routing.</Callout>}

          {enabled && (
            <div className="studio-exposure-form">
              <FormGroup label="Canonical MCP tool name" labelFor="mcp-tool-name" required helperText={`System-wide public identity. Studio requires the owner prefix ${report?.ownerPackage ?? ''}; Cube and CubeCompose names are reserved automatically.`}>
                <InputGroup
                  id="mcp-tool-name"
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                  onBlur={() => setName(canonicalMCPName(name))}
                  disabled={saving}
                  autoComplete="off"
                />
              </FormGroup>
              <FormGroup label="Description" labelFor="mcp-description" helperText={`${description.trim().length}/512 characters`}>
                <InputGroup
                  id="mcp-description"
                  value={description}
                  onChange={(event) => setDescription(event.target.value)}
                  disabled={saving}
                />
              </FormGroup>
              <FormGroup label="Description resource" labelFor="mcp-description-path" helperText="Optional relative path to a versioned documentation resource. Publication fails if the resource is unavailable.">
                <InputGroup
                  id="mcp-description-path"
                  placeholder="docs/vendor-summary.md"
                  value={descriptionPath}
                  onChange={(event) => setDescriptionPath(event.target.value)}
                  disabled={saving}
                  autoComplete="off"
                />
              </FormGroup>
            </div>
          )}

          </section>

        </DialogBody>
        <DialogFooter actions={(
          <>
            <Button onClick={onClose} disabled={saving}>Cancel</Button>
            <Button type="submit" intent="primary" icon="floppy-disk" loading={saving} disabled={!route || loadingConnectors}>
              Save component
            </Button>
          </>
        )} />
      </form>
    </Dialog>
  );
}
