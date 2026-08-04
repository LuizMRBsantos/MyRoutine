import { useEffect, useState } from 'react';
import type { Model, Query } from '@nozbe/watermelondb';

/**
 * Assina uma query do WatermelonDB e re-renderiza quando os dados mudam.
 * Preferido ao HOC withObservables por manter as telas como componentes de
 * função simples.
 *
 * `deps` controla quando a query é reassinada — passe os valores que a
 * constroem (datas, filtros), nunca o objeto de query.
 */
export function useObservableQuery<T extends Model>(
  buildQuery: () => Query<T>,
  deps: unknown[] = []
): T[] {
  const [records, setRecords] = useState<T[]>([]);

  useEffect(() => {
    const subscription = buildQuery().observe().subscribe(setRecords);
    return () => subscription.unsubscribe();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  return records;
}
