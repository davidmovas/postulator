import { describe, expect, it } from "vitest";

import { localAddress } from "./address.js";

describe("localAddress", () => {
    it.each([
        ["http://localhost:8089", true],
        ["http://127.0.0.1:8089/wp", true],
        ["http://127.5.4.3", true],
        ["http://[::1]:8089", true],
        ["http://10.0.0.7", true],
        ["http://172.16.4.1", true],
        ["http://172.32.4.1", false],
        ["http://192.168.1.10:8080", true],
        ["http://169.254.10.4", true],
        ["http://[fd00::1]:8080", true],
        ["http://shop.local", true],
        ["http://app.localhost", true],
        ["http://wordpress.test", true],
        ["http://SHOP.LOCAL", true],
        ["https://example.com", false],
        ["http://example.com:8080", false],
        ["http://93.184.216.34", false],
        ["http://localhost.example.com", false],
        ["shop.local", false],
        ["", false],
        ["   ", false],
        ["not a url", false],
    ])("reads %s as %s", (raw, want) => {
        expect(localAddress(raw)).toBe(want);
    });
});
