export interface DatasetField {
    description?: string;
    hidden: boolean;
    name: string;
    type: string;
    unit: string;
}
export interface DatasetFields {

    datasetName: string;
    fields: DatasetField[];
}

export interface OrderedColumn {
    Name: string;
    Type: string;
    CslType: string;
    DocString?: string;
}

export interface Table {
    Name: string;
    DocString?: string;
    OrderedColumns: OrderedColumn[];
}

export interface Database {
    Name: string;
    Tables: { [key: string]: Table };
    // FUTURE: Functions
    Functions: { [key: string]: any };
}

export interface MonacoKustoSchema {
    Plugins: any;
    Databases: { [key: string]: Database };
}

export function mapDatasetInfosToSchema(datasetInfos: DatasetFields[]): MonacoKustoSchema | undefined {
    const tables: { [key: string]: Table } = {};

    datasetInfos.forEach((datasetInfo) => {
        const { datasetName, fields } = datasetInfo;

        const columns: OrderedColumn[] = [];

        (fields || []).forEach((field) => {
            if (field.hidden) {
                return;
            }

            const { name, type } = field;

            let columnType = columnTypeForAxiomType(type);
            if (name === '_time' || name === '_sysTime') {
                columnType = { Type: 'System.DateTime', CslType: 'datetime' };
            }

            if (!columnType) {
                console.warn('Unable to map field:', name, type);
            } else {
                columns.push({
                    Name: name,
                    ...columnType,
                });
            }
        });

        tables[datasetName] = {
            Name: datasetName,
            OrderedColumns: columns,
        };
    });

    return {
        Plugins: [],
        Databases: {
            db: {
                Name: 'db',
                Tables: tables,
                Functions: {},
            },
        },
    };
}

const columnTypeForAxiomType = (axiomType: string): { Type: string; CslType: string } | undefined => {
    // 'System.Int32': 'int',
    // 'System.String': 'string',
    // 'System.Single': 'float',
    // 'System.Boolean': 'bool',
    // 'Newtonsoft.Json.Linq.JArray': 'dynamic',
    const types = axiomType.split('|');

    if (types.length > 1) {
        if (axiomType === 'integer|float') {
            return { Type: 'System.Double', CslType: 'real' };
        }

        return { Type: 'System.Object', CslType: 'dynamic' };
    }

    switch (types[0]) {
        case 'integer':
            return { Type: 'System.Int64', CslType: 'long' };
        case 'float':
            return { Type: 'System.Double', CslType: 'real' };
        case 'datetime':
            return { Type: 'System.DateTime', CslType: 'datetime' };
        case 'timestamp':
            return { Type: 'System.DateTime', CslType: 'datetime' };
        case 'string':
            return { Type: 'System.String', CslType: 'string' };
        case 'boolean':
            return { Type: 'System.Boolean', CslType: 'bool' };
        case 'array':
            return { Type: 'Newtonsoft.Json.Linq.JArray', CslType: 'dynamic' };
        case 'map':
            return { Type: 'Newtonsoft.Json.Linq.JObject', CslType: 'dynamic' };
    }

    return undefined;
};
