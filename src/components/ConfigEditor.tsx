import React, { ChangeEvent } from 'react';
import { InlineField, SecretInput, Input } from '@grafana/ui';
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
      <InlineField label="Host" labelWidth={12}>
        <Input
          onChange={onHostChange}
          value={jsonData.apiHost || 'https://api.axiom.co'}
          placeholder="axiom host"
          width={40}
        />
      </InlineField>
      <InlineField label="API Token" labelWidth={12}>
        <SecretInput
          isConfigured={(secureJsonFields && secureJsonFields.accessToken) as boolean}
          value={secureJsonData.accessToken || ''}
          placeholder="xaat-***********"
          width={40}
          onReset={onResetAccessToken}
          onChange={onAccessTokenChange}
        />
      </InlineField>
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
