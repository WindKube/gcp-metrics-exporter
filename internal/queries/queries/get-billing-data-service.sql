SELECT
  invoice.month AS billing_month,
  COALESCE(service.description, 'unknown') AS service,
  'all' AS cost_type,
  currency,
  CAST(SUM(cost + IFNULL((SELECT SUM(credit.amount) FROM UNNEST(credits) AS credit), 0)) AS STRING) AS cost
FROM `{{TABLE}}`
WHERE project.id = @project_id
  AND invoice.month >= @invoice_month_start
GROUP BY billing_month, service, currency
ORDER BY billing_month, service, cost_type, currency
