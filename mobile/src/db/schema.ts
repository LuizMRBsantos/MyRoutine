import { appSchema, tableSchema } from '@nozbe/watermelondb';

export default appSchema({
  version: 1,
  tables: [
    tableSchema({
      name: 'notes',
      columns: [
        { name: 'date', type: 'string', isIndexed: true },
        { name: 'content', type: 'string' },
        { name: 'metadata', type: 'string', isOptional: true },
        { name: 'is_collection', type: 'boolean' },
        { name: 'created_at', type: 'number' },
        { name: 'updated_at', type: 'number' },
      ]
    }),
    tableSchema({
      name: 'transactions',
      columns: [
        { name: 'value', type: 'number' },
        { name: 'category', type: 'string', isIndexed: true },
        { name: 'method', type: 'string', isOptional: true },
        { name: 'date', type: 'string', isIndexed: true },
        { name: 'note_id', type: 'string', isIndexed: true },
        { name: 'created_at', type: 'number' },
        { name: 'updated_at', type: 'number' },
      ]
    }),
    tableSchema({
      name: 'health_records',
      columns: [
        { name: 'type', type: 'string', isIndexed: true },
        { name: 'distance', type: 'number', isOptional: true },
        { name: 'time', type: 'number', isOptional: true },
        { name: 'rpe', type: 'number', isOptional: true },
        { name: 'macros', type: 'string', isOptional: true },
        { name: 'date', type: 'string', isIndexed: true },
        { name: 'note_id', type: 'string', isIndexed: true },
        { name: 'created_at', type: 'number' },
        { name: 'updated_at', type: 'number' },
      ]
    })
  ]
});
