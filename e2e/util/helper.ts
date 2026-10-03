import { Page, expect } from "@playwright/test";

const uiURL = process.env.UI_URL ? process.env.UI_URL : "http://localhost:8080";

export async function login(
  page: Page,
  email: string,
  password: string,
): Promise<void> {
  await page.goto(uiURL + "/ui/login/");
  await expect(page).toHaveURL(/login\/$/);
  await page.getByPlaceholder("you@company.com").fill(email);
  await page
    .locator("form[name='password-login'] input[type='password']")
    .fill(password);
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
}

// Cancels all bookings of the currently logged-in user via the API.
// Used to make tests independent of leftover bookings from a previous
// failed/retried run (e.g. a different browser project sharing the same
// backend within one test run).
export async function cancelAllBookings(page: Page): Promise<void> {
  const accessToken = await page.evaluate(() =>
    window.localStorage.getItem("accessToken"),
  );
  const headers = { Authorization: `Bearer ${accessToken}` };
  const res = await page.request.get(uiURL + "/booking/", { headers });
  const bookings = await res.json();
  for (const booking of bookings) {
    await page.request.delete(uiURL + "/booking/" + booking.id, { headers });
  }
}

/**
 * Provisions a fresh client of the platform and signs in as them.
 *
 * In this fork the bootstrap account is the platform operator: they run the
 * service, have no workspace of their own, and so cannot exercise the desks,
 * bookings and areas these tests are about. A client is the equivalent of the
 * organization administrator upstream logs in as, so one is created per test.
 *
 * A client set up by the operator must choose their own password on first
 * sign-in, which this drives through the form the login page shows.
 */
export async function loginAsNewClient(page: Page): Promise<string> {
  const suffix = Math.random().toString().substr(2, 8);
  const email = `client${suffix}@example.test`;
  const initialPassword = "Sea!surf1ng";
  const password = "Cl!ent" + suffix;

  const org = await page.request.get(uiURL + "/organization/domain/localhost");
  const organizationId = (await org.json()).id;
  const auth = await page.request.post(uiURL + "/auth/login", {
    data: {
      email: "admin@seatsurfing.local",
      password: "Sea!surf1ng",
      organizationId,
    },
  });
  const headers = {
    Authorization: `Bearer ${(await auth.json()).accessToken}`,
  };

  // The client's directory record, then the workspace they administer.
  const created = await page.request.post(uiURL + "/user/", {
    headers,
    data: {
      email,
      password: initialPassword,
      firstname: "E2E",
      lastname: "Client" + suffix,
    },
  });
  const clientId = created.headers()["x-object-id"];
  const createdOrg = await page.request.post(uiURL + "/organization/", {
    headers,
    data: {
      name: "E2E Org " + suffix,
      firstname: "E2E",
      lastname: "Client" + suffix,
      email,
      language: "en",
    },
  });
  await page.request.post(
    uiURL + "/user/" + clientId + "/organizations",
    {
      headers,
      data: { organizationId: createdOrg.headers()["x-object-id"] },
    },
  );

  await login(page, email, initialPassword);

  // First sign-in: the operator set the password, so the client sets their own.
  await expect(page.locator("form[name='password-update']")).toBeVisible();
  await page
    .locator("form[name='password-update'] input[type='password']")
    .fill(password);
  await page
    .locator("form[name='password-update'] button[type='submit']")
    .click();

  return email;
}
