import { schemaMigrations, addColumns } from '@nozbe/watermelondb/Schema/migrations';

// Migrações locais do WatermelonDB — não confundir com as migrações SQL do
// backend. Colunas novas sempre opcionais ou com valor preenchido no código,
// para não invalidar linhas já gravadas no aparelho.
export default schemaMigrations({
  migrations: [
    {
      toVersion: 2,
      steps: [
        addColumns({
          table: 'transactions',
          columns: [
            { name: 'description', type: 'string', isOptional: true },
            { name: 'push_status', type: 'string', isIndexed: true },
            { name: 'pushed_at', type: 'number', isOptional: true },
          ],
        }),
        addColumns({
          table: 'health_records',
          columns: [
            { name: 'push_status', type: 'string', isIndexed: true },
            { name: 'pushed_at', type: 'number', isOptional: true },
          ],
        }),
      ],
    },
  ],
});
