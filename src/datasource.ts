import {
  DataQueryRequest,
  DataQueryResponse,
  DataSourceApi,
  DataSourceInstanceSettings,
  MutableDataFrame,
  FieldType,
} from '@grafana/data';
import { getBackendSrv } from '@grafana/runtime';
import { lastValueFrom } from 'rxjs';

import { AxiomQuery, AxiomDataSourceOptions, APLResponse } from './types';

export class DataSource extends DataSourceApi<AxiomQuery, AxiomDataSourceOptions> {
  url?: string;

  constructor(instanceSettings: DataSourceInstanceSettings<AxiomDataSourceOptions>) {
    super(instanceSettings);
    this.url = instanceSettings.url;
  }

  async query(options: DataQueryRequest<AxiomQuery>): Promise<DataQueryResponse> {
    console.log(options);
    // data frames: https://grafana.com/docs/grafana/latest/developers/plugins/data-frames/
    const { range } = options;
    const from = range!.raw.from.toString();
    const to = range!.raw.to.valueOf().toString();

    const promises = options.targets.map(async (target) => {
      const resp = await this.doRequest({ ...target, startTime: from, endTime: to });
      console.log(resp);

      let frame: MutableDataFrame;

      const table = resp?.data?.tables[0]!;
      const fields = table.fields.map((f: any, index: number) => {
        return { name: f.name, type: resolveFieldType(f.type), values: table.columns[index] };
      });
      frame = new MutableDataFrame({
        name: table.name,
        refId: target.refId,
        fields,
      });

      return frame;
    });

    return Promise.all(promises).then((data) => ({ data }));
  }

  async doRequest(query: AxiomQuery) {
    if (!query.apl) {
      return;
    }
    const resp = getBackendSrv().fetch<APLResponse>({
      method: 'POST',
      url: `${this.url}/datasets/_apl`,
      data: query,
      params: {
        format: 'tabular',
      },
    });

    return lastValueFrom(resp);
  }

  async testDatasource() {
    return await getBackendSrv().post(
      `${this.url}/datasets/_apl`,
      { apl: "['vercel'] | limit 1", startTime: 'now-7d', endTime: 'now' },
      {
        params: {
          format: 'tabular',
        },
      }
    );
  }
}

const resolveFieldType = (axiomFieldType: string) => {
  switch (axiomFieldType) {
    case 'datetime':
      return FieldType.time;
    case 'integer':
      return FieldType.number;
    default:
      return FieldType.string;
  }
};
