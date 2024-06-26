// Core Grafana history https://github.com/grafana/grafana/blob/v11.0.0-preview/public/app/plugins/datasource/prometheus/configuration/mocks.ts
import { DataSourceSettings } from '@grafana/data';

import { getMockDataSource } from '../gcopypaste/app/features/datasources/__mocks__/dataSourcesMocks';
import { PromOptions } from '../types';

export function createDefaultConfigOptions(): DataSourceSettings<PromOptions> {
  return getMockDataSource<PromOptions>({
    jsonData: {
      timeInterval: '1m',
      queryTimeout: '1m',
      httpMethod: 'GET',
    },
  });
}
