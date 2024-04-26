import React, { ChangeEvent } from 'react';
import { InlineField, SecretInput, Input, Label } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { AxiomDataSourceOptions, MySecureJsonData } from '../types';

interface Props extends DataSourcePluginOptionsEditorProps<AxiomDataSourceOptions, MySecureJsonData> {}

export function ConfigEditor(props: Props) {
  const { onOptionsChange, options } = props;
  const onHostChange = (event: ChangeEvent<HTMLInputElement>) => {
    const jsonData = {
      ...options.jsonData,
      apiHost: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };

  const onOrgIDChange = (event: ChangeEvent<HTMLInputElement>) => {
    const jsonData = {
      ...options.jsonData,
      orgID: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };

  // Secure field (only sent to the backend)
  const onAccessTokenChange = (event: ChangeEvent<HTMLInputElement>) => {
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

  const { secureJsonFields } = options;
  const jsonData = (options.jsonData || {}) as AxiomDataSourceOptions;
  const secureJsonData = (options.secureJsonData || {}) as MySecureJsonData;

  return (
    <div className="gf-form-group">
      <Label description={<span>Create a Personal Token from your Axiom account settings.</span>}>
        <h5>Authentication</h5>
      </Label>
      <InlineField label="Personal Token" labelWidth={17}>
        <SecretInput
          isConfigured={(secureJsonFields && secureJsonFields.accessToken) as boolean}
          value={secureJsonData.accessToken || ''}
          placeholder="xapt-***********"
          width={40}
          onReset={onResetAccessToken}
          onChange={onAccessTokenChange}
        />
      </InlineField>
      <br />
      <InlineField label="Org ID" labelWidth={17}>
        <Input value={jsonData.orgID || ''} placeholder="" width={40} onChange={onOrgIDChange} />
      </InlineField>
      <br />
      <div>
        <Label description="The Axiom host to use.">
          <h6>Axiom Host</h6>
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
