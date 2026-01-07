// src/utils/ipv6.ts

/** 将完整 IPv6 压缩成最短文本 */
export function compressIPv6(full: string): string {
  // 1. 拆成 8 段 16 bit
  const segs = full.split(':').map(s => parseInt(s, 16));

  // 2. 先压缩前导 0
  const zTrim = segs.map(n => n.toString(16));

  // 3. 找最长连续全 0 段
  let bestStart = -1, bestLen = 0;
  for (let i = 0; i < segs.length; ) {
    if (segs[i] === 0) {
      let j = i + 1;
      while (j < segs.length && segs[j] === 0) j++;
      const len = j - i;
      if (len > bestLen) { bestLen = len; bestStart = i; }
      i = j;
    } else i++;
  }

  // 4. 拼接，把最长 0 段换成 ::
  const out: string[] = [];
  for (let i = 0; i < segs.length; i++) {
    if (i === bestStart && bestLen > 0) {
      out.push('');               // 用空串占位，一会替换成 ::
      i += bestLen - 1;
      continue;
    }
    out.push(zTrim[i]);
  }

  // 5. 合并并处理边界情况
  let ans = out.join(':').replace(/:{2,}/, '::');
  // 头尾出现 :: 时可能多冒号，再修一次
  ans = ans.replace(/^:/, '::').replace(/:$/, '::');
  return ans.toLowerCase();
}

/* ====== 使用示例 ====== */
// import { compressIPv6 } from '@/utils/ipv6';
// compressIPv6('2001:0db8:0000:0000:0000:ff00:0042:8329')
// → "2001:db8::ff00:42:8329"