import {
  DataQueryRequest,
  DataQueryResponse,
  DataSourceApi,
  DataSourceInstanceSettings,
  MutableDataFrame,
  FieldType,
} from '@grafana/data';
import { getBackendSrv } from '@grafana/runtime';

import { AxiomQuery, AxiomDataSourceOptions } from './types';

export class DataSource extends DataSourceApi<AxiomQuery, AxiomDataSourceOptions> {
  url?: string;

  constructor(instanceSettings: DataSourceInstanceSettings<AxiomDataSourceOptions>) {
    super(instanceSettings);
    this.url = instanceSettings.url;
  }

  async query(options: DataQueryRequest<AxiomQuery>): Promise<DataQueryResponse> {
    // data frames: https://grafana.com/docs/grafana/latest/developers/plugins/data-frames/
    const { range } = options;
    const from = range!.raw.from.toString();
    const to = range!.raw.to.valueOf().toString();

    const promises = await options.targets.map(async (target) => {
      const resp = await this.doRequest({ ...target, startTime: from, endTime: to });

      let frame: MutableDataFrame;
      //  = new MutableDataFrame({
      //   refId: target.refId,
      //   fields: [
      //     { name: 'time', type: FieldType.time },
      //     { name: 'value', type: FieldType.number },
      //     // { name: 'id', type: FieldType.string },
      //   ],
      // });


      await resp?.forEach((point: any) => {
        const table = point.data.tables[1];
        const fields = table.fields.map((f: any, index: number) => {
          return { name: f.name, type: resolveFieldType(f.type), values: table.columns[index] }
        })
        frame = new MutableDataFrame({
          name: table.name,
          refId: target.refId,
          fields,
        })
      });
      // legacy format series
      // resp?.forEach((point: any) => {
      //   point.data.buckets.series.forEach((series: any) => {
      //     const time = new Date(series.startTime);
      //     series.groups?.forEach((g: any) => {
      //       frame.add({time: time.getTime(), value: g.aggregations[0].value});
      //     });
      //   });
      // });

      return frame;
    });


    return Promise.all(promises).then((data) => ({ data }));
  }

  async doRequest(query: AxiomQuery) {
    if (!query.apl) {
      return;
    }
    const resp = await getBackendSrv().fetch({
      method: 'POST',
      url: `${this.url}/datasets/_apl`,
      data: query,
      params: {
        format: 'tabular',
      },
    });

    return resp;
  }

  async testDatasource() {
    return await getBackendSrv().post(
      `${this.url}/datasets/_apl`,
      { apl: "['vercel'] | limit 1", startTime: 'now-7d', endTime: 'now' },
      {
        params: {
          format: 'legacy',
        },
      }
    );
  }
}

const resolveFieldType = (axiomFieldType: string) => {
  switch (axiomFieldType) {
    case 'datetime':
      return FieldType.time
    case 'integer':
      return FieldType.number
    default:
      return FieldType.string
  }
}
