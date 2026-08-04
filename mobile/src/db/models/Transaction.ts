import { Model } from '@nozbe/watermelondb';
import { field, date, readonly, relation, text } from '@nozbe/watermelondb/decorators';
import type Note from './Note';

export default class Transaction extends Model {
  static table = 'transactions';

  @field('value') value!: number;
  @text('category') category!: string;
  @text('method') method?: string;
  @text('date') date!: string;

  @relation('notes', 'note_id') note!: Note;

  @readonly @date('created_at') createdAt!: Date;
  @readonly @date('updated_at') updatedAt!: Date;
}
