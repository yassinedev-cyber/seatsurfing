import { test, expect } from "@playwright/test";
import { login } from "../util/helper";

// The platform operator creates the client, and the client always comes with a
// workspace to administer - without one they would sign in to nothing. Further
// workspaces are added afterwards, and an existing one can be attached.
//
// Organizations carry no domain of their own: every client signs in at the
// platform's single address and is resolved from their email, so none of these
// flows asks for one.

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

/** Fills and submits the new-client form, returning once it has been saved. */
async function createClient(page: any, firstname: string, email: string) {
  await page.getByRole("link", { name: "Add" }).click();
  await expect(page).toHaveURL(/admin\/clients\/add\/$/);

  await page.locator("#form input[type='text']").nth(0).fill(firstname);
  await page.locator("#form input[type='text']").nth(1).fill("Client");
  await page.locator("#form input[type='email']").fill(email);
  await page.locator("#form input[type='password']").fill("Sea!surf1ng");
  await page.getByRole("button", { name: "Save" }).click();

  await expect(page).toHaveURL(/admin\/clients\/[0-9a-f-]{36}\/$/);
}

test("a new client comes with a workspace named after them", async ({
  page,
}) => {
  const suffix = Math.random().toString().substr(2, 6);

  await createClient(page, "Test" + suffix, `client${suffix}@example.test`);

  // Naming the workspace is optional, so it takes the client's own name.
  await expect(
    page.getByRole("link", { name: `Test${suffix} Client` }),
  ).toBeVisible();
});

test("a client can be given a second workspace", async ({ page }) => {
  const suffix = Math.random().toString().substr(2, 6);
  const orgName = "Org " + suffix;

  await createClient(page, "Second", `client${suffix}@example.test`);

  await page.locator("#form-add-org input[type='text']").first().fill(orgName);
  await page.getByRole("button", { name: "Add organization" }).click();
  await expect(page.getByRole("link", { name: orgName })).toBeVisible();
});

test("detaching an organization keeps the organization itself", async ({
  page,
}) => {
  const suffix = Math.random().toString().substr(2, 6);
  const orgName = "Org " + suffix;

  await createClient(page, "Later", `client${suffix}@example.test`);
  await page.locator("#form-add-org input[type='text']").first().fill(orgName);
  await page.getByRole("button", { name: "Add organization" }).click();
  await expect(page.getByRole("link", { name: orgName })).toBeVisible();

  // Detaching withdraws this client's access; the workspace is not deleted.
  page.once("dialog", (dialog) => dialog.accept());
  await page
    .locator("tr", { hasText: orgName })
    .getByRole("button", { name: "Detach" })
    .click();
  await expect(page.getByRole("link", { name: orgName })).toHaveCount(0);

  // It is still there - listed with no client and nobody in it - and so can be
  // attached again.
  await page.getByRole("link", { name: "Organizations", exact: true }).click();
  await expect(page).toHaveURL(/admin\/organizations\/$/);
  await expect(
    page.locator("tr", { hasText: orgName }).getByRole("cell").first(),
  ).toHaveText(orgName);
});

test("update client and its organization", async ({ page }) => {
  const suffix = Math.random().toString().substr(2, 6);
  const orgName = "Org " + suffix;

  await createClient(page, "Update", `client${suffix}@example.test`);
  await page.locator("#form-add-org input[type='text']").first().fill(orgName);
  await page.getByRole("button", { name: "Add organization" }).click();
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
  await page
    .locator("#form input[type='text']")
    .first()
    .fill(orgName + " SARL");
  await page.getByRole("button", { name: "Save" }).click();
  await expect(page.getByText("Record saved.")).toBeVisible();
  await page.reload();
  await expect(page.locator("#form input[type='text']").first()).toHaveValue(
    orgName + " SARL",
  );
});
