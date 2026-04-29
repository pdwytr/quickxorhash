import subprocess
import argparse


def calc_hash(metadata_path, download_path):
    """
    Updates file timestamps using a PowerShell script.

    Args:
        metadata_path (str): Path to the CSV file containing file paths and timestamps.
        download_path (str): Base path to prepend to the extracted destination paths.

    Returns:
        tuple: (status_code: bool, error: str)
            - status_code: True if successful, False if an error occurred.
            - error: Error message if an error occurred, otherwise None.
    """
    try:
        # PowerShell script to read the CSV, process paths, and update timestamps
        powershell_script = f"""
        $filesData = Import-Csv -Path "{metadata_path}"

        foreach ($row in $filesData) {{
            # Extract the part of the destination path after the colon
            $source = $row.source
            # assuming the 
            $extractedPath = $source -replace '^[^:]*:', ''
            $fileName = $row.name
            # Combine download folder path with the relative path and filename
            $fullPath = Join-Path -Path (Join-Path -Path "{download_path}" -ChildPath $extractedPath) -ChildPath $fileName
            Write-Host "Processing file: $fullPath"
            # Run QuickXorHash and capture the output
            $hashResult = cmd /c .\quickxorhash.exe $fullPath 2`>`&1
            if ($hashResult) {{
                # Add the hash to the row
                $row | Add-Member -MemberType NoteProperty -Name 'calculated_hash' -Value $hashResult -Force
            }}
            $filesData | Export-Csv -Path {metadata_path} -NoTypeInformation -Force
        }}
        """

        # Run the PowerShell script
        result = subprocess.run(
            ["powershell", "-Command", powershell_script],
            capture_output=True,
            text=True,
            check=True,
        )

        # Print the output for debugging
        print("PowerShell Output:")
        print(result.stdout)

        # If no exceptions, return success
        return True, None

    except subprocess.CalledProcessError as e:
        # Handle errors from PowerShell
        error_message = f"PowerShell Error: {e.stderr}"
        print(error_message)
        return False, error_message

    except Exception as e:
        # Handle any other exceptions
        error_message = f"Unexpected Error: {str(e)}"
        print(error_message)
        return False, error_message


def parse_arguments():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--metadata",
        default=os.path.join(os.getcwd(), "outputs/metadata.csv"),
        help="Absolute path to the manifest file",
    )
    parser.add_argument(
        "--download",
        default=os.path.join(os.getcwd(), "downloads"),
        help="Absolute path to the download folder",
    )
    return parser.parse_args()


def main():
    # args = parse_arguments()
    metadata_path = input("Enter the path to the metadata file: ")
    download_path = input("Enter the path to the root dir: ")
    success, error = calc_hash(metadata_path, download_path)
    if not success:
        print(f"Failed to process files: {error}")
        return 1
    return 0


if __name__ == "__main__":
    exit(main())
