import { test, expect } from "@playwright/test";
import { login } from "../util/helper";

// The platform operator creates the client first. The organization they own can
// be created in the same step, or added and attached at any later point.

test.beforeEach(async ({ page }) => {
  // Suppress the MFA encouragement modal
  await page.addInitScript(() => {
    window.localStorage.setItem("mfaEncouragementDismissed", "1");
  });

  await login(page, "admin@seatsurfing.local", "Sea!surf1ng");

  // A platform operator has no workspace of their own: they land on their
  // client overview rather than on a booking page.
  await expect(page).toHaveURL(/admin\/overview\/$/);

  await page.getByRole("link", { name: "Clients", exact: true }).click();
  await expect(page).toHaveURL(/admin\/clients\/$/);
});

test("create client together with an organization", async ({ page }) => {
  const suffix = Math.random().toString().substr(2, 6);
  const email = `client${suffix}@example.test`;
  const orgName = "Org " + suffix;

  await page.getByRole("link", { name: "Add" }).click();
  await expect(page).toHaveURL(/admin\/clients\/add\/$/);

  await page.locator("#form input[type='text']").nth(0).fill("Test");
  await page.locator("#form input[type='text']").nth(1).fill("Client");
  await page.locator("#form input[type='email']").fill(email);
  await page.locator("#form input[type='password']").fill("Sea!surf1ng");

  // The organization is requested by default
  await expect(page.locator("#withOrg")).toBeChecked();
  await page.locator("#form input[type='text']").nth(2).fill(orgName);
  await page.getByPlaceholder("acme.codyn.se").fill(`org${suffix}.codyn.test`);
  await page.getByRole("button", { name: "Save" }).click();

  // The page switches to the saved client and lists the new organization
  await expect(page).toHaveURL(/admin\/clients\/[0-9a-f-]{36}\/$/);
  await expect(page.getByRole("link", { name: orgName })).toBeVisible();
});

test("create client and attach an organization later", async ({ page }) => {
  const suffix = Math.random().toString().substr(2, 6);
  const email = `client${suffix}@example.test`;
  const orgName = "Org " + suffix;

  await page.getByRole("link", { name: "Add" }).click();
  await page.locator("#form input[type='text']").nth(0).fill("Later");
  await page.locator("#form input[type='text']").nth(1).fill("Client");
  await page.locator("#form input[type='email']").fill(email);
  await page.locator("#form input[type='password']").fill("Sea!surf1ng");
  // the organization is offered by default; this flow adds it afterwards
  await page.locator("#withOrg").uncheck();
  await page.getByRole("button", { name: "Save" }).click();

  // No organization yet
  await expect(page).toHaveURL(/admin\/clients\/[0-9a-f-]{36}\/$/);
  await expect(page.getByText("No organization attached yet.")).toBeVisible();

  // Add one after the fact
  await page.locator("#form-add-org input[type='text']").first().fill(orgName);
  await page.getByPlaceholder("acme.codyn.se").fill(`org${suffix}.codyn.test`);
  await page.getByRole("button", { name: "Add organization" }).click();
  await expect(page.getByRole("link", { name: orgName })).toBeVisible();

  // And detach it again - the organization itself is kept
  page.once("dialog", (dialog) => dialog.accept());
  await page
    .locator("tr", { hasText: orgName })
    .getByRole("button", { name: "Detach" })
    .click();
  await expect(page.getByText("No organization attached yet.")).toBeVisible();
});

test("update client and its organization", async ({ page }) => {
  const suffix = Math.random().toString().substr(2, 6);
  const email = `client${suffix}@example.test`;
  const orgName = "Org " + suffix;

  await page.getByRole("link", { name: "Add" }).click();
  await page.locator("#form input[type='text']").nth(0).fill("Update");
  await page.locator("#form input[type='text']").nth(1).fill("Client");
  await page.locator("#form input[type='email']").fill(email);
  await page.locator("#form input[type='password']").fill("Sea!surf1ng");
  await page.locator("#form input[type='text']").nth(2).fill(orgName);
  await page.getByPlaceholder("acme.codyn.se").fill(`org${suffix}.codyn.test`);
  await page.getByRole("button", { name: "Save" }).click();
  await expect(page.getByRole("link", { name: orgName })).toBeVisible();

  // Rename the client
  await page.locator("#form input[type='text']").nth(1).fill("Renamed");
  await page.getByRole("button", { name: "Save" }).click();
  // wait for the save to land before reloading, or the request gets cancelled
  await expect(page.getByText("Record saved.")).toBeVisible();
  await page.reload();
  await expect(page.locator("#form input[type='text']").nth(1)).toHaveValue(
    "Renamed",
  );

  // Rename the organization from the client's organization list
  await page.getByRole("link", { name: orgName }).click();
  await expect(page).toHaveURL(/admin\/organizations\/[0-9a-f-]{36}\/$/);
  await page.locator("#form input[type='text']").first().fill(orgName + " SARL");
  await page.getByRole("button", { name: "Save" }).click();
  await expect(page.getByText("Record saved.")).toBeVisible();
  await page.reload();
  await expect(page.locator("#form input[type='text']").first()).toHaveValue(
    orgName + " SARL",
  );
});
