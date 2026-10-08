import { describe, it, expect } from 'vitest'
import { urlBase64ToUint8Array } from './push'

describe('urlBase64ToUint8Array', () => {
  it('decodifica base64url sem padding', () => {
    // "-_8" em base64url = 0xFB 0xFF ("+/8=" em base64 comum)
    expect(Array.from(urlBase64ToUint8Array('-_8'))).toEqual([0xfb, 0xff])
    expect(Array.from(urlBase64ToUint8Array('AQID'))).toEqual([1, 2, 3])
  })
})
