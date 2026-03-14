# data-analyst

Skill for analyzing datasets using Python and pandas.

## When to Use This Skill

Use this skill when:
- User asks to analyze CSV, JSON, or Excel data files
- User needs statistical summaries or visualizations
- User wants to clean or transform data
- User requests data exploration or pattern detection

## Instructions

1. Load the data file using pandas (read_csv, read_json, or read_excel)
2. Display first 5 rows with df.head() and check shape
3. Check for missing values with df.isnull().sum()
4. Generate descriptive statistics with df.describe()
5. Create visualizations using matplotlib or seaborn as needed
6. Document findings in a clear summary

## Tools

- pandas - data manipulation
- matplotlib - plotting
- seaborn - statistical visualizations
- numpy - numerical operations

## Install

```
pip install pandas matplotlib seaborn numpy
```

## Commands

### data-analyst load

Loads a data file and displays basic information.

**Flags:**
- `--format json` — output as JSON

**JSON output:**
```json
{
  "rows": 100,
  "columns": 5,
  "missing": {"col1": 0, "col2": 3}
}
```

**Exit codes:**
- 0: file loaded successfully
- 1: file not found or invalid format

### data-analyst summarize

Generates statistical summary of the data.

**Flags:**
- `--format json` — output as JSON

**JSON output:**
```json
{
  "mean": [1.5, 2.3],
  "std": [0.5, 0.8],
  "min": [0.1, 1.0],
  "max": [3.0, 5.0]
}
```

**Exit codes:**
- 0: summary generated
- 1: no data loaded

## What this does NOT do

- Does not modify the original data files
- Does not upload data to external services
- Does not execute arbitrary code from untrusted sources

## Parsing examples

```bash
data-analyst load --format json | jq '.rows'
data-analyst summarize --format json | jq '.mean'
```
