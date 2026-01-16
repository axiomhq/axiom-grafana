import React, { ChangeEvent, useState } from 'react';
import { InlineField, SecretInput, Input, Label, Alert } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { AxiomDataSourceOptions, MySecureJsonData } from '../types';

interface Props extends DataSourcePluginOptionsEditorProps<AxiomDataSourceOptions, MySecureJsonData> {}

export function ConfigEditor(props: Props) {
  const { onOptionsChange, options } = props;
  const [shouldShowOrgId, setShowOrgId] = useState(
    !!options.jsonData.orgID && options.secureJsonData?.accessToken.startsWith('xapt-')
  );

  const onHostChange = (event: ChangeEvent<HTMLInputElement>) => {
    const jsonData = {
      ...options.jsonData,
      apiHost: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };

  const onRegionChange = (event: ChangeEvent<HTMLInputElement>) => {
    const jsonData = {
      ...options.jsonData,
      region: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };

  // Secure field (only sent to the backend)
  const onAccessTokenChange = (event: ChangeEvent<HTMLInputElement>) => {
    if (event.target.value.startsWith('xapt-')) {
      setShowOrgId(true);
    } else {
      setShowOrgId(false);
    }

    onOptionsChange({
      ...options,
      secureJsonData: {
        accessToken: event.target.value,
      },
    });
  };

  const onResetAccessToken = () => {
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        accessToken: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        accessToken: '',
      },
    });
  };

  const onOrgIDChange = (event: ChangeEvent<HTMLInputElement>) => {
    const jsonData = {
      ...options.jsonData,
      orgID: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };

  const { secureJsonFields } = options;
  const jsonData = (options.jsonData || {}) as AxiomDataSourceOptions;
  const secureJsonData = (options.secureJsonData || {}) as MySecureJsonData;

  return (
    <div className="gf-form-group">
      <Label description={<span>Create an API Token from your Axiom account settings.</span>}>
        <h5>Authentication</h5>
      </Label>
      <InlineField label="API Token" labelWidth={17}>
        <SecretInput
          isConfigured={(secureJsonFields && secureJsonFields.accessToken) as boolean}
          value={secureJsonData.accessToken || ''}
          placeholder="xaat-***********"
          width={40}
          onReset={onResetAccessToken}
          onChange={onAccessTokenChange}
        />
      </InlineField>
      <br />
      {/* Only show orgId for users who have already set it. Promote advanced tokens instead */}
      {shouldShowOrgId && (
        <InlineField label="Org ID" labelWidth={17}>
          <Input value={jsonData.orgID || ''} placeholder="" width={40} onChange={onOrgIDChange} />
        </InlineField>
      )}
      {/* If orgId is set, show a deprecation message */}
      {shouldShowOrgId && (
        <div>
          <Alert
            title="Personal tokens are deprecated and will be removed in the next release. Please switch to advanced API tokens."
            about="Token"
            severity="warning"
            buttonContent="Learn more"
            topSpacing={4}
          />
        </div>
      )}
      <div>
        <Label description="The Axiom API host for management operations (schema lookup, health checks).">
          <h6>Axiom API Host</h6>
        </Label>
        <InlineField label="URL" labelWidth={17}>
          <Input
            onChange={onHostChange}
            value={jsonData.apiHost || 'https://api.axiom.co'}
            placeholder="Axiom API host URL"
            width={40}
          />
        </InlineField>
      </div>
      <div>
        <Label description="Optional regional edge domain for queries. When set, queries are routed to this edge. Leave empty to use the API host for queries.">
          <h6>Edge Region (Optional)</h6>
        </Label>
        <InlineField label="Region" labelWidth={17}>
          <Input
            onChange={onRegionChange}
            value={jsonData.region || ''}
            placeholder="e.g., eu-central-1.aws.edge.axiom.co"
            width={40}
          />
        </InlineField>
      </div>
    </div>
  );
}

export function isValid(settings: AxiomDataSourceOptions): boolean {
  if (!settings) {
    return false;
  }

  const { apiHost } = settings;
  if (!apiHost) {
    return false;
  }

  return true;
}
