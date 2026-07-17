export interface ClipboardWriter {
  writeText(text: string): Promise<void>;
}

export function tableInviteText(inviteCode: string): string {
  return `邀请你加入 DePu 房间，邀请码：${inviteCode}`;
}

export async function copyTableInvite(inviteCode: string, clipboard?: ClipboardWriter | null): Promise<string> {
  try {
    if (!clipboard) throw new Error('clipboard unavailable');
    await clipboard.writeText(tableInviteText(inviteCode));
    return `邀请码已复制：${inviteCode}`;
  } catch {
    return `邀请码：${inviteCode}`;
  }
}
