import { describe, expect, it } from 'vitest';
import { isAllowedApiPath, isSameOrigin } from './policy';
describe('Границы BFF', () => {
  it('отклоняет запись без Origin и с чужим Origin', () => {
    for (const origin of [
      undefined,
      'null',
      'https://evil.example',
      'https://app.example.evil.test',
    ]) {
      const request = new Request('https://app.example/api/v1/tasks', {
        method: 'POST',
        headers: origin ? { origin } : {},
      });
      expect(isSameOrigin(request, 'https://app.example')).toBe(false);
    }
    expect(
      isSameOrigin(
        new Request('https://app.example/api/v1/tasks', {
          headers: { origin: 'https://app.example' },
        }),
        'https://app.example',
      ),
    ).toBe(true);
  });
  it('разрешает только версионные API-пути без обхода каталогов и кодированных разделителей', () => {
    for (const path of [
      [],
      ['v2', 'tasks'],
      ['v1', '..'],
      ['v1', '%2fadmin'],
      ['v1', 'a/b'],
      ['v1', 'https:'],
      ['v1', ''],
    ])
      expect(isAllowedApiPath(path)).toBe(false);
    expect(
      isAllowedApiPath(['v1', 'tasks', '123e4567-e89b-12d3-a456-426614174000']),
    ).toBe(true);
  });
});
