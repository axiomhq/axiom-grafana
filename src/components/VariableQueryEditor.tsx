import { DataSource } from 'datasource';
import React from 'react';
import { QueryEditor } from './QueryEditor';
import { AxiomQuery } from '../types';

interface Props {
    query: AxiomQuery;
    onChange: (query: AxiomQuery, definition?: string) => void;
    datasource: DataSource;
}

export const VariableQueryEditor = ({ onChange, query, datasource }: Props) => {
    const saveQuery = (newQuery: AxiomQuery) => {
        if (newQuery) {
            onChange(newQuery, newQuery.apl);
        }
    };

    return (
        <QueryEditor
            onRunQuery={() => {}}
            onChange={saveQuery}
            query={{ ...query }}
            datasource={datasource}
        />
    );
};
