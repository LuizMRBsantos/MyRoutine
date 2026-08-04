import { Model } from '@nozbe/watermelondb';
import { field, date, readonly, relation, text } from '@nozbe/watermelondb/decorators';
import type Note from './Note';

export default class Transaction extends Model {
  static table = 'transactions';

  @field('value') value!: number;
  @text('description') description?: string;
  @text('category') category!: string;
  @text('method') method?: string;
  @text('date') date!: string;

  // 'pending' | 'synced' — o push para a API é unidirecional e idempotente
  // (o servidor deduplica por source_type='track_day' + source_id = este id).
  @text('push_status') pushStatus!: string;
  @field('pushed_at') pushedAt?: number;

  @relation('notes', 'note_id') note!: Note;

  @readonly @date('created_at') createdAt!: Date;
  @readonly @date('updated_at') updatedAt!: Date;
}
