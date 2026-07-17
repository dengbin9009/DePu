import { describe, expect, it, vi } from 'vitest';
import { copyTableInvite } from './tableInvite';

describe('copyTableInvite', () => {
  it('copies invitation text containing the room invite code', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);

    const notice = await copyTableInvite('ABC123', { writeText });

    expect(writeText).toHaveBeenCalledOnce();
    expect(writeText).toHaveBeenCalledWith(expect.stringContaining('ABC123'));
    expect(notice).toBe('邀请码已复制：ABC123');
  });

  it('returns the invite code for manual copying when clipboard writing fails', async () => {
    const writeText = vi.fn().mockRejectedValue(new Error('clipboard unavailable'));

    const notice = await copyTableInvite('ABC123', { writeText });

    expect(notice).toBe('邀请码：ABC123');
  });
});
